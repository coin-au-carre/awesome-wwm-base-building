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
	dryRun := flag.Bool("dry-run", false, "fetch posts but skip writing JSON")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	channelID := cmdutil.RequireEnv("WHATEVER_SHOWCASE_CHANNEL_ID")
	guildID := cmdutil.RequireEnv("DISCORD_GUILD_ID")

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating discord session", "err", err)
		os.Exit(1)
	}
	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	posts, err := discord.FetchWhateverShowcase(s, channelID, guildID)
	if err != nil {
		slog.Error("fetching whatever showcase", "err", err)
		os.Exit(1)
	}

	slog.Info("found posts with images", "count", len(posts))

	if *dryRun {
		slog.Info("dry-run: skipping write")
		return
	}

	out, err := json.MarshalIndent(posts, "", "  ")
	if err != nil {
		slog.Error("marshalling posts", "err", err)
		os.Exit(1)
	}

	dest := filepath.Join(*root, "data", "whatever.json")
	if err := os.WriteFile(dest, out, 0o644); err != nil {
		slog.Error("writing whatever.json", "err", err)
		os.Exit(1)
	}

	slog.Info("wrote whatever.json", "path", dest, "posts", len(posts))
}
