// cmd/sync/main.go
package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"fmt"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"
	"ruby/internal/guild"

	"github.com/joho/godotenv"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "crawl Discord but skip writing JSON files")
	noNotify := flag.Bool("no-notify", false, "skip posting summary to bot channel")
	root := flag.String("root", cmdutil.RootDir(), "root directory containing guilds.json and solos.json")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	guildForumID := cmdutil.RequireEnv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")
	soloForumID := os.Getenv("SOLO_BUILD_SHOWCASE_CHANNEL_FORUM_ID")
	botChannelID := os.Getenv("BOT_CHANNEL_ID")
	devChannelID := os.Getenv("DEV_CHANNEL_ID")
	generalChannelID := os.Getenv("GENERAL_CHANNEL_ID")
	baseBuilderRoleID := os.Getenv("BASE_BUILDER_ROLE_ID")
	baseCriticRoleID := os.Getenv("BASE_CRITIC_ROLE_ID")

	guildsPath    := filepath.Join(*root, "data", "guilds.json")
	solosPath     := filepath.Join(*root, "data", "solos.json")
	roleCachePath := filepath.Join(*root, "data", "role_assignments.json")

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


	// ── Fetch both channels in parallel ──────────────────────────────────────

	type fetchOutcome struct {
		result discord.SyncFetchResult
		err    error
	}

	guildCh := make(chan fetchOutcome, 1)
	soloCh := make(chan fetchOutcome, 1)

	var fetchWg sync.WaitGroup
	fetchWg.Add(1)
	go func() {
		defer fetchWg.Done()
		r, err := discord.SyncFetch(bot, guilds, discord.SyncConfig{ForumChannelID: guildForumID})
		guildCh <- fetchOutcome{r, err}
	}()

	if soloForumID != "" {
		fetchWg.Add(1)
		go func() {
			defer fetchWg.Done()
			r, err := discord.SyncFetch(bot, solos, discord.SyncConfig{ForumChannelID: soloForumID, IsSolo: true})
			soloCh <- fetchOutcome{r, err}
		}()
	}

	fetchWg.Wait()
	close(guildCh)
	close(soloCh)

	guildFetch := <-guildCh
	if guildFetch.err != nil {
		bot.NotifyIf(!*noNotify, "💥 **Guilds failed to synchronize!** — "+guildFetch.err.Error())
		slog.Error("guild fetch failed", "err", guildFetch.err)
		os.Exit(1)
	}

	var soloFetch fetchOutcome
	if soloForumID != "" {
		soloFetch = <-soloCh
		if soloFetch.err != nil {
			bot.NotifyIf(!*noNotify, "💥 **Solo construction failed to synchronize!** — "+soloFetch.err.Error())
			slog.Error("solo fetch failed", "err", soloFetch.err)
			os.Exit(1)
		}
	}

	// ── Merge voter counts → cross-channel weights ────────────────────────────

	mergedCounts := discord.MergeVoterCounts(guildFetch.result.VoterCounts, soloFetch.result.VoterCounts)
	mergedWeights := discord.ComputeVoterWeights(mergedCounts)
	slog.Info("merged voter weights", "voters", len(mergedWeights))

	// ── Finalize (score) both channels ────────────────────────────────────────

	updatedGuilds, guildStats := discord.SyncFinalize(guildFetch.result, mergedWeights)

	var updatedSolos []guild.Guild
	var soloStats discord.SyncStats
	if soloForumID != "" {
		updatedSolos, soloStats = discord.SyncFinalize(soloFetch.result, mergedWeights)
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

	}

	// ── Role assignment ───────────────────────────────────────────────────────

	if !*dryRun && (baseBuilderRoleID != "" || baseCriticRoleID != "") {
		forumCh, err := bot.Session.Channel(guildForumID)
		if err != nil {
			slog.Warn("fetching forum channel for role assignment", "err", err)
		} else {
			roleCache, err := discord.LoadRoleCache(roleCachePath)
			if err != nil {
				slog.Warn("loading role cache, skipping cache", "err", err)
				roleCache = nil
			}
			discordGuildID := forumCh.GuildID
			if baseBuilderRoleID != "" {
				discord.AssignRoleByScore(bot.Session, discordGuildID, baseBuilderRoleID, updatedGuilds, 0, nil, roleCache)
				if soloForumID != "" {
					discord.AssignRoleByScore(bot.Session, discordGuildID, baseBuilderRoleID, updatedSolos, 0, nil, roleCache)
				}
			}
			if baseCriticRoleID != "" {
				slog.Info("assigning critic role", "total_voters", len(mergedCounts))
				discord.AssignRoleToVoters(bot.Session, discordGuildID, baseCriticRoleID, mergedCounts, 6, roleCache)
			}
			if err := roleCache.Save(); err != nil {
				slog.Warn("saving role cache", "err", err)
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

	if !*dryRun {
		const maxNewAnnouncements = 5
		guildSpam := guildStats.New > maxNewAnnouncements
		soloSpam := soloStats.New > maxNewAnnouncements

		if guildSpam || soloSpam {
			warn := fmt.Sprintf(
				"⚠️ **Announcement suppressed** — unusually high new-entry count (guilds: %d, solos: %d). Check that guilds.json loaded correctly before announcing.",
				guildStats.New, soloStats.New,
			)
			slog.Warn("announcement suppressed: too many new entries", "guilds_new", guildStats.New, "solos_new", soloStats.New)
			if !*noNotify {
				bot.NotifyIf(true, warn)
				if devChannelID != "" {
					bot.Send(devChannelID, warn)
				}
			}
		} else {
			if !*noNotify && guildStats.New > 0 {
				notifyNewEntries(bot, updatedGuilds, guildStats, false)
			}
			if !*noNotify && soloStats.New > 0 {
				notifyNewEntries(bot, updatedSolos, soloStats, true)
			}
			if !*noNotify && generalChannelID != "" {
				announceToGeneral(bot, generalChannelID, updatedGuilds, guildStats, false)
				announceToGeneral(bot, generalChannelID, updatedSolos, soloStats, true)
			}
		}
	}

	if devChannelID != "" && !*noNotify {
		allWarnings := append(guildStats.DuplicateWarnings, soloStats.DuplicateWarnings...)
		for _, w := range allWarnings {
			bot.Send(devChannelID, w)
		}
	}
}

const (
	ahlyamID = "149790526076354561"
	windxpID = "721510597958828183"
	babeID   = "376950312721711118"
)

func announceToGeneral(bot *discord.Bot, channelID string, entries []guild.Guild, stats discord.SyncStats, isSolo bool) {
	byName := make(map[string]guild.Guild, len(entries))
	for _, g := range entries {
		byName[g.Name] = g
	}
	announce := func(name string, isSoloEntry bool, msgFn func(guild.Guild, bool) string) {
		g, ok := byName[name]
		if !ok {
			return
		}
		bot.Send(channelID, msgFn(g, isSoloEntry))
	}
	for _, name := range stats.NewNames {
		g, ok := byName[name]
		if !ok || g.BuilderDiscordID == ahlyamID || g.BuilderDiscordID == windxpID || g.BuilderDiscordID == babeID {
			continue
		}
		msg := discord.FormatNewGuildMessage(g, isSolo)
		if len(g.Screenshots) > 0 {
			imgData, filename, err := discord.DownloadImage(g.Screenshots[0])
			if err == nil {
				bot.SendWithFile(channelID, msg, filename, imgData)
				imgData.Close()
				continue
			}
		}
		bot.Send(channelID, msg)
	}
	for _, name := range stats.MoreScreenshotNames {
		g, ok := byName[name]
		if !ok || g.BuilderDiscordID == ahlyamID {
			continue
		}
		if g.BuilderDiscordID == babeID && g.Name != "PleasureSeeker" {
			continue
		}
		announce(name, isSolo, discord.FormatMoreScreenshotsMessage)
	}
	for _, name := range stats.MoreVideoNames {
		g, ok := byName[name]
		if !ok || g.BuilderDiscordID == ahlyamID {
			continue
		}
		if g.BuilderDiscordID == babeID && g.Name != "PleasureSeeker" {
			continue
		}
		announce(name, isSolo, discord.FormatMoreVideosMessage)
	}
}

func notifyNewEntries(bot *discord.Bot, entries []guild.Guild, stats discord.SyncStats, isSolo bool) {
	byName := make(map[string]guild.Guild, len(entries))
	for _, g := range entries {
		byName[g.Name] = g
	}
	for _, name := range stats.NewNames {
		g, ok := byName[name]
		if !ok || g.BuilderDiscordID == ahlyamID || g.BuilderDiscordID == windxpID || g.BuilderDiscordID == babeID {
			continue
		}
		bot.Notify(discord.FormatNewGuildMessage(g, isSolo))
	}
}

