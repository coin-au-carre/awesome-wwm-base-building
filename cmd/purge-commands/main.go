package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"ruby/internal/cmdutil"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	root := flag.String("root", cmdutil.RootDir(), "root directory")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	guildID := os.Getenv("DISCORD_GUILD_ID")

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating session", "err", err)
		os.Exit(1)
	}

	// Need the app ID — open briefly to populate State.
	if err := s.Open(); err != nil {
		slog.Error("opening session", "err", err)
		os.Exit(1)
	}
	appID := s.State.User.ID
	s.Close()

	deleteAll := func(scope string) {
		label := "global"
		if scope != "" {
			label = "guild:" + scope
		}
		cmds, err := s.ApplicationCommands(appID, scope)
		if err != nil {
			slog.Error("listing commands", "scope", label, "err", err)
			return
		}
		slog.Info("found commands", "scope", label, "count", len(cmds))
		for _, c := range cmds {
			slog.Info("deleting", "scope", label, "name", c.Name, "id", c.ID)
			if err := s.ApplicationCommandDelete(appID, scope, c.ID); err != nil {
				slog.Error("deleting command", "name", c.Name, "err", err)
			}
		}
	}

	deleteAll("")       // global
	deleteAll(guildID)  // guild-scoped

	slog.Info("done — restart the bot to re-register commands")
}
