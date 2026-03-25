package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

const DRY_RUN = false
const DO_NOT_NOTIFY = false

type SyncStats struct {
	Total        int
	Updated      int
	New          int
	UpdatedNames []string
	NewNames     []string
}

func rootDir() string {
	if _, err := os.Stat("LICENSE"); err == nil {
		return "."
	}
	return ".."
}

func main() {
	noNotify := flag.Bool("no-notify", false, "disable Discord notifications")
	flag.Parse()

	root := rootDir()
	if err := godotenv.Load(filepath.Join(root, ".env")); err != nil {
		slog.Warn("no .env file, relying on environment variables")
	}

	guildBaseShowcaseChannelForumID := os.Getenv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")
	if guildBaseShowcaseChannelForumID == "" {
		slog.Error("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID not set")
		os.Exit(1)
	}

	botChannelID := os.Getenv("BOT_CHANNEL_ID")
	if botChannelID == "" {
		slog.Error("BOT_CHANNEL_ID not set")
		os.Exit(1)
	}

	session, err := discordgo.New("Bot " + os.Getenv("RUBY_BOT_TOKEN"))
	if err != nil {
		slog.Error("creating session", "err", err)
		os.Exit(1)
	}
	defer session.Close()

	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent

	if err := session.Open(); err != nil {
		slog.Error("opening session", "err", err)
		os.Exit(1)
	}

	stats, err := syncGuilds(session, root, guildBaseShowcaseChannelForumID)
	if err != nil {
		notify(session, botChannelID, "💥 **Guilds have failed to synchronize grrr!** — "+err.Error(), noNotify)
		slog.Error("sync failed", "err", err)
		os.Exit(1)
	}

	notify(session, botChannelID, formatSyncSummary(stats), noNotify)
}
