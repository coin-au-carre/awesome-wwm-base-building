package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"ruby/internal/cmdutil"
)

func main() {
	channel := flag.String("channel", "", "discord channel ID to post in (overrides BOT_CHANNEL_ID)")
	user := flag.String("user", "", "discord user ID to DM directly")
	image := flag.String("image", "", "path to an image file to attach")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(cmdutil.RootDir(), ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	if *user != "" && *channel != "" {
		slog.Error("use -user or -channel, not both")
		os.Exit(1)
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating session", "err", err)
		os.Exit(1)
	}

	channelID := *channel
	if *user != "" {
		ch, err := s.UserChannelCreate(*user)
		if err != nil {
			slog.Error("opening DM channel", "user", *user, "err", err)
			os.Exit(1)
		}
		channelID = ch.ID
	} else if channelID == "" {
		channelID = cmdutil.RequireEnv("RUBY_CHANNEL_ID")
	}

	text := flag.Arg(0)
	if text == "" && *image == "" {
		slog.Error("provide a message and/or -image path")
		os.Exit(1)
	}

	if *image != "" {
		f, err := os.Open(*image)
		if err != nil {
			slog.Error("opening image", "err", err)
			os.Exit(1)
		}
		defer f.Close()

		_, err = s.ChannelFileSendWithMessage(channelID, text, filepath.Base(*image), f)
		if err != nil {
			slog.Error("sending image", "err", err)
			os.Exit(1)
		}
	} else {
		if _, err := s.ChannelMessageSend(channelID, text); err != nil {
			slog.Error("sending message", "err", err)
			os.Exit(1)
		}
	}

	slog.Info("sent", "channel", channelID)
}
