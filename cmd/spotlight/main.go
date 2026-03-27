package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	idiscord "ruby/internal/discord"
	"ruby/internal/guild"
)

func main() {
	if err := godotenv.Load(filepath.Join(rootDir(), ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := requireEnv("RUBY_BOT_TOKEN")
	channelID := requireEnv("RUBY_CHANNEL_ID")

	guilds, err := guild.Load(rootDir())
	if err != nil {
		slog.Error("loading guilds", "err", err)
		os.Exit(1)
	}

	pick, imgURL, ok := idiscord.PickRandomGuild(guilds)
	if !ok {
		slog.Error("no guilds with screenshots found")
		os.Exit(1)
	}

	msg := idiscord.FormatSpotlightMessage(pick, true)

	imgData, filename, err := idiscord.DownloadImage(imgURL)
	if err != nil {
		slog.Error("downloading screenshot", "err", err, "url", imgURL)
		os.Exit(1)
	}
	defer imgData.Close()

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating discord session", "err", err)
		os.Exit(1)
	}

	if _, err = s.ChannelFileSendWithMessage(channelID, msg, filename, imgData); err != nil {
		slog.Error("sending spotlight", "err", err)
		os.Exit(1)
	}

	slog.Info("spotlight sent", "guild", pick.Name, "channel", channelID)
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
