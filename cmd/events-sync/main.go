// cmd/events-sync/main.go — fetch Discord Scheduled Events → data/events.json
package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	root := flag.String("root", cmdutil.RootDir(), "root directory")
	dryRun := flag.Bool("dry-run", false, "fetch events but skip writing JSON")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	guildID := cmdutil.RequireEnv("DISCORD_GUILD_ID")

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating discord session", "err", err)
		os.Exit(1)
	}

	events, err := discord.FetchEvents(s, guildID)
	if err != nil {
		slog.Error("fetching events", "err", err)
		os.Exit(1)
	}

	slog.Info("fetched events", "count", len(events))

	if *dryRun {
		slog.Info("dry-run: skipping write")
		return
	}

	out, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		slog.Error("marshalling events", "err", err)
		os.Exit(1)
	}

	dest := filepath.Join(*root, "data", "events.json")
	if err := os.WriteFile(dest, out, 0o644); err != nil {
		slog.Error("writing events.json", "err", err)
		os.Exit(1)
	}

	slog.Info("wrote events.json", "path", dest)
}
