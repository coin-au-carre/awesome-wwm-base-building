// cmd/announce/main.go — post a guild/solo announcement card to a Discord channel.
// Useful for testing the general-channel announcement format before running a full sync.
package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"ruby/internal/cmdutil"
	idiscord "ruby/internal/discord"
	"ruby/internal/guild"
)

func main() {
	guildName := flag.String("guild", "", "guild/solo name to announce (substring match, required)")
	channelID := flag.String("channel", "", "Discord channel ID to post in (default: BOT_CHANNEL_ID)")
	isSolo := flag.Bool("solo", false, "search in solos instead of guilds")
	screenshots := flag.Bool("screenshots", false, "use the 'new screenshots' message format instead of 'new entry'")
	flag.Parse()

	if *guildName == "" {
		slog.Error("provide a guild name with -guild")
		os.Exit(1)
	}

	if err := godotenv.Load(filepath.Join(cmdutil.RootDir(), ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	target := *channelID
	if target == "" {
		target = os.Getenv("GENERAL_CHANNEL_ID")
	}
	if target == "" {
		slog.Error("provide -channel or set GENERAL_CHANNEL_ID")
		os.Exit(1)
	}

	root := cmdutil.RootDir()
	dataPath := filepath.Join(root, "data", "guilds.json")
	if *isSolo {
		dataPath = filepath.Join(root, "data", "solos.json")
	}

	entries, err := guild.LoadFile(dataPath)
	if err != nil {
		slog.Error("loading data", "path", dataPath, "err", err)
		os.Exit(1)
	}

	pick, _, ok := idiscord.PickGuildByName(entries, *guildName)
	if !ok {
		slog.Error("no entry matching name", "name", *guildName)
		os.Exit(1)
	}

	var msg string
	if *screenshots {
		msg = idiscord.FormatMoreScreenshotsMessage(pick, *isSolo)
	} else {
		msg = idiscord.FormatNewGuildMessage(pick, *isSolo)
	}

	bot, err := idiscord.NewBot(token, "")
	if err != nil {
		slog.Error("creating discord session", "err", err)
		os.Exit(1)
	}

	if len(pick.Screenshots) > 0 {
		if *screenshots {
			msg += "\n" + pick.Screenshots[len(pick.Screenshots)-1]
		} else {
			msg += "\n" + pick.Screenshots[0]
		}
	}
	bot.Send(target, msg)
	slog.Info("announced", "guild", pick.Name, "channel", target)
}
