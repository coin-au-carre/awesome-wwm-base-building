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
	dryRun := flag.Bool("dry-run", false, "crawl Discord but skip writing guilds.json")
	noNotify := flag.Bool("no-notify", false, "skip posting summary to bot channel")
	root := flag.String("root", rootDir(), "root directory containing guilds.json")
	forceRole := flag.Bool("force-role", false, "reassign Base Builder role to all thread authors, including already-known ones")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := requireEnv("RUBY_BOT_TOKEN")
	forumChannelID := requireEnv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")
	soloForumChannelID := os.Getenv("SOLO_BUILD_SHOWCASE_CHANNEL_FORUM_ID")
	botChannelID := os.Getenv("BOT_CHANNEL_ID")
	baseBuilderRoleID := os.Getenv("BASE_BUILDER_ROLE_ID")

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

	guilds, err := guild.Load(*root)
	if err != nil {
		slog.Warn("could not load guilds, starting fresh", "err", err)
		guilds = []guild.Guild{}
	}

	// Build skip-set of already-known authors before sync adds new ones.
	var knownAuthors map[string]bool
	if !*forceRole {
		knownAuthors = make(map[string]bool, len(guilds))
		for _, g := range guilds {
			if g.BuilderDiscordID != "" {
				knownAuthors[g.BuilderDiscordID] = true
			}
		}
	}

	updated, stats, err := discord.Sync(bot, guilds, discord.SyncConfig{
		ForumChannelID: forumChannelID,
		DryRun:         *dryRun,
	})
	if err != nil {
		bot.NotifyIf(!*noNotify, "💥 **Guilds failed to synchronize!** — "+err.Error())
		slog.Error("sync failed", "err", err)
		os.Exit(1)
	}

	if *dryRun {
		slog.Info("dry-run: skipping save")
	} else {
		if err := guild.Save(*root, updated); err != nil {
			slog.Error("saving guilds", "err", err)
			os.Exit(1)
		}
	}

	if !*dryRun && baseBuilderRoleID != "" {
		for _, ch := range []string{forumChannelID, soloForumChannelID} {
			if ch == "" {
				continue
			}
			if err := discord.AssignRoleToForumAuthors(bot.Session, ch, baseBuilderRoleID, knownAuthors); err != nil {
				slog.Warn("assigning base builder roles", "channel", ch, "err", err)
			}
		}
	}

	bot.NotifyIf(!*noNotify, discord.FormatSyncSummary(stats))
	slog.Info("sync complete", "total", stats.Total, "new", stats.New, "updated", stats.Updated)
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
