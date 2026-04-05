package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"ruby/internal/cmdutil"
	idiscord "ruby/internal/discord"
	"ruby/internal/guild"
)

func main() {
	if err := godotenv.Load(filepath.Join(cmdutil.RootDir(), ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	channelID := cmdutil.RequireEnv("RUBY_CHANNEL_ID")

	guildName := flag.String("guild", "", "spotlight a specific guild by name (substring match)")
	flag.Parse()

	guilds, err := guild.Load(cmdutil.RootDir())
	if err != nil {
		slog.Error("loading guilds", "err", err)
		os.Exit(1)
	}

	var pick guild.Guild
	var imgURL string
	var ok bool
	random := *guildName == ""
	if random {
		pick, imgURL, ok = idiscord.PickRandomGuild(guilds)
	} else {
		pick, imgURL, ok = idiscord.PickGuildByName(guilds, *guildName)
	}
	if !ok {
		if random {
			slog.Error("no guilds with screenshots found")
		} else {
			slog.Error("no guild matching name", "name", *guildName)
		}
		os.Exit(1)
	}

	msg := idiscord.FormatSpotlightMessage(pick, random)

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
