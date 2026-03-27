package main

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

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

	var candidates []guild.Guild
	for _, g := range guilds {
		if len(g.Screenshots) > 0 {
			candidates = append(candidates, g)
		}
	}
	if len(candidates) == 0 {
		slog.Error("no guilds with screenshots found")
		os.Exit(1)
	}

	pick := candidates[rand.IntN(len(candidates))]
	imgURL := pick.Screenshots[rand.IntN(len(pick.Screenshots))]

	msg := formatMessage(pick)

	imgData, filename, err := downloadImage(imgURL)
	if err != nil {
		slog.Error("downloading screenshot", "err", err, "url", imgURL)
		os.Exit(1)
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating discord session", "err", err)
		os.Exit(1)
	}

	_, err = s.ChannelFileSendWithMessage(channelID, msg, filename, imgData)
	if err != nil {
		slog.Error("sending spotlight", "err", err)
		os.Exit(1)
	}

	slog.Info("spotlight sent", "guild", pick.Name, "channel", channelID)
}

func formatMessage(g guild.Guild) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## 🏰 Guild Spotlight: **%s**\n", g.Name)
	fmt.Fprintf(&sb, "-# 🎲 Randomly picked from the list\n")
	if len(g.Builders) > 0 {
		fmt.Fprintf(&sb, "👷 Built by: %s\n", strings.Join(g.Builders, ", "))
	}
	if len(g.Tags) > 0 {
		fmt.Fprintf(&sb, "🏷️ %s\n", strings.Join(g.Tags, ", "))
	}
	fmt.Fprintf(&sb, "⭐ Score: %d\n", g.Score)
	if g.DiscordThread != "" {
		fmt.Fprintf(&sb, "🔗 %s", g.DiscordThread)
	}
	return sb.String()
}

func downloadImage(url string) (io.Reader, string, error) {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("HTTP %d fetching image", resp.StatusCode)
	}
	// derive a filename from the URL path
	parts := strings.Split(strings.Split(url, "?")[0], "/")
	name := parts[len(parts)-1]
	if name == "" {
		name = "screenshot.png"
	}
	return resp.Body, name, nil
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
