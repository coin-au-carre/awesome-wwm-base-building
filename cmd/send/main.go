package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	channel := flag.String("channel", "", "discord channel ID to post in (overrides BOT_CHANNEL_ID)")
	image := flag.String("image", "", "path to an image file to attach")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(rootDir(), ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := requireEnv("RUBY_BOT_TOKEN")

	channelID := *channel
	if channelID == "" {
		channelID = requireEnv("RUBY_CHANNEL_ID")
	}

	text := flag.Arg(0)
	if text == "" && *image == "" {
		slog.Error("provide a message and/or -image path")
		os.Exit(1)
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating session", "err", err)
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
