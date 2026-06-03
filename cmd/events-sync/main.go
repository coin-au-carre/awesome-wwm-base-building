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
)

func main() {
	root := flag.String("root", cmdutil.RootDir(), "root directory")
	dryRun := flag.Bool("dry-run", false, "fetch events but skip writing JSON")
	flag.Parse()

	cmdutil.LoadEnv(*root)

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

	dest := filepath.Join(*root, "data", "events.json")

	// Check existing IDs before overwriting, so we can detect new events.
	existingIDs := map[string]bool{}
	if raw, err := os.ReadFile(dest); err == nil {
		var existing []discord.Event
		if json.Unmarshal(raw, &existing) == nil {
			for _, e := range existing {
				existingIDs[e.ID] = true
			}
		}
	}

	out, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		slog.Error("marshalling events", "err", err)
		os.Exit(1)
	}

	if err := os.WriteFile(dest, append(out, '\n'), 0o644); err != nil {
		slog.Error("writing events.json", "err", err)
		os.Exit(1)
	}

	slog.Info("wrote events.json", "path", dest)

	for _, e := range events {
		if !existingIDs[e.ID] {
			cmdutil.UpdateNavVersion(*root, "events")
			break
		}
	}
}
