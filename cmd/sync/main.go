// cmd/sync/main.go
package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"ruby/internal/discord"
	"ruby/internal/guild"

	"github.com/joho/godotenv"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "crawl Discord but skip writing JSON files")
	noNotify := flag.Bool("no-notify", false, "skip posting summary to bot channel")
	root := flag.String("root", rootDir(), "root directory containing guilds.json and solos.json")
	forceRole := flag.Bool("force-role", false, "reassign roles to all thread authors, including already-known ones")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := requireEnv("RUBY_BOT_TOKEN")
	guildForumID := requireEnv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")
	soloForumID := os.Getenv("SOLO_BUILD_SHOWCASE_CHANNEL_FORUM_ID")
	botChannelID := os.Getenv("BOT_CHANNEL_ID")
	baseBuilderRoleID := os.Getenv("BASE_BUILDER_ROLE_ID")
	baseCriticRoleID := os.Getenv("BASE_CRITIC_ROLE_ID")

	guildsPath := filepath.Join(*root, "guilds.json")
	solosPath := filepath.Join(*root, "solos.json")

	bot, err := discord.NewBot(token, botChannelID)
	if err != nil {
		slog.Error("creating bot", "err", err)
		os.Exit(1)
	}
	defer bot.Close()

	if err := bot.Open(); err != nil {
		slog.Error("opening session", "err", err)
		os.Exit(1)
	}

	// ── Load existing data ────────────────────────────────────────────────────

	guilds, err := guild.LoadFile(guildsPath)
	if err != nil {
		slog.Warn("could not load guilds, starting fresh", "err", err)
		guilds = []guild.Guild{}
	}

	var solos []guild.Guild
	if soloForumID != "" {
		solos, err = guild.LoadFile(solosPath)
		if err != nil {
			slog.Warn("could not load solos, starting fresh", "err", err)
			solos = []guild.Guild{}
		}
	}

	// Build skip-sets of already-known authors before sync adds new ones.
	var knownGuildAuthors, knownSoloAuthors map[string]bool
	if !*forceRole {
		knownGuildAuthors = knownAuthorSet(guilds)
		knownSoloAuthors = knownAuthorSet(solos)
	}

	// ── Collect voter counts from both channels → merged weights ─────────────
	// Voters who reacted across guild AND solo threads get a combined count,
	// so e.g. 4 guild votes + 2 solo votes = 6 → 2× weight.

	slog.Info("collecting voter counts from guild channel")
	guildVoterCounts, err := discord.CollectVoterCounts(bot, guildForumID)
	if err != nil {
		slog.Warn("collecting guild voter counts", "err", err)
		guildVoterCounts = map[string]int{}
	}

	soloVoterCounts := map[string]int{}
	if soloForumID != "" {
		slog.Info("collecting voter counts from solo channel")
		soloVoterCounts, err = discord.CollectVoterCounts(bot, soloForumID)
		if err != nil {
			slog.Warn("collecting solo voter counts", "err", err)
			soloVoterCounts = map[string]int{}
		}
	}

	mergedCounts := discord.MergeVoterCounts(guildVoterCounts, soloVoterCounts)
	mergedWeights := discord.ComputeVoterWeights(mergedCounts)
	slog.Info("merged voter weights", "voters", len(mergedWeights))

	// ── Sync guild channel ────────────────────────────────────────────────────

	updatedGuilds, guildStats, err := discord.Sync(bot, guilds, discord.SyncConfig{
		ForumChannelID:       guildForumID,
		DryRun:               *dryRun,
		ExternalVoterWeights: mergedWeights,
	})
	if err != nil {
		bot.NotifyIf(!*noNotify, "💥 **Guilds failed to synchronize!** — "+err.Error())
		slog.Error("guild sync failed", "err", err)
		os.Exit(1)
	}

	// ── Sync solo channel ─────────────────────────────────────────────────────

	var updatedSolos []guild.Guild
	var soloStats discord.SyncStats
	if soloForumID != "" {
		updatedSolos, soloStats, err = discord.Sync(bot, solos, discord.SyncConfig{
			ForumChannelID:       soloForumID,
			DryRun:               *dryRun,
			ExternalVoterWeights: mergedWeights,
		})
		if err != nil {
			bot.NotifyIf(!*noNotify, "💥 **Solo builds failed to synchronize!** — "+err.Error())
			slog.Error("solo sync failed", "err", err)
			os.Exit(1)
		}
	}

	// ── Save ──────────────────────────────────────────────────────────────────

	if *dryRun {
		slog.Info("dry-run: skipping save")
	} else {
		if err := guild.SaveFile(guildsPath, updatedGuilds); err != nil {
			slog.Error("saving guilds", "err", err)
			os.Exit(1)
		}
		if soloForumID != "" {
			if err := guild.SaveFile(solosPath, updatedSolos); err != nil {
				slog.Error("saving solos", "err", err)
				os.Exit(1)
			}
		}

		if !*noNotify && guildStats.New > 0 {
			notifyNewEntries(bot, updatedGuilds, guildStats, false)
		}
		if !*noNotify && soloStats.New > 0 {
			notifyNewEntries(bot, updatedSolos, soloStats, true)
		}
	}

	// ── Role assignment ───────────────────────────────────────────────────────

	if !*dryRun && (baseBuilderRoleID != "" || baseCriticRoleID != "") {
		forumCh, err := bot.Session.Channel(guildForumID)
		if err != nil {
			slog.Warn("fetching forum channel for role assignment", "err", err)
		} else {
			discordGuildID := forumCh.GuildID
			if baseBuilderRoleID != "" {
				discord.AssignRoleByScore(bot.Session, discordGuildID, baseBuilderRoleID, updatedGuilds, 0, knownGuildAuthors)
				if soloForumID != "" {
					discord.AssignRoleByScore(bot.Session, discordGuildID, baseBuilderRoleID, updatedSolos, 0, knownSoloAuthors)
				}
			}
			if baseCriticRoleID != "" {
				// Use merged counts: voters across both channels are credited together
				slog.Info("assigning critic role", "total_voters", len(mergedCounts))
				discord.AssignRoleToVoters(bot.Session, discordGuildID, baseCriticRoleID, mergedCounts, 6)
			}
		}
	}

	// ── Summary ───────────────────────────────────────────────────────────────

	bot.NotifyIf(!*noNotify, discord.FormatSyncSummary(guildStats))
	slog.Info("guild sync complete", "total", guildStats.Total, "new", guildStats.New, "updated", guildStats.Updated)
	if soloForumID != "" {
		bot.NotifyIf(!*noNotify, discord.FormatSoloSyncSummary(soloStats))
		slog.Info("solo sync complete", "total", soloStats.Total, "new", soloStats.New, "updated", soloStats.Updated)
	}
}

func notifyNewEntries(bot *discord.Bot, entries []guild.Guild, stats discord.SyncStats, isSolo bool) {
	byName := make(map[string]guild.Guild, len(entries))
	for _, g := range entries {
		byName[g.Name] = g
	}
	for _, name := range stats.NewNames {
		g, ok := byName[name]
		if !ok {
			continue
		}
		msg := discord.FormatNewGuildMessage(g, isSolo)
		if len(g.Screenshots) > 0 {
			imgData, filename, err := discord.DownloadImage(g.Screenshots[0])
			if err == nil {
				bot.NotifyWithFile(msg, filename, imgData)
				imgData.Close()
				continue
			}
		}
		bot.Notify(msg)
	}
}

func knownAuthorSet(guilds []guild.Guild) map[string]bool {
	m := make(map[string]bool, len(guilds))
	for _, g := range guilds {
		if g.BuilderDiscordID != "" {
			m[g.BuilderDiscordID] = true
		}
	}
	return m
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env variable not set", "key", key)
		os.Exit(1)
	}
	return v
}

func rootDir() string {
	if _, err := os.Stat("LICENSE"); err == nil {
		return "."
	}
	return ".."
}
