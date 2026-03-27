package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"ruby/internal/discord"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	root := flag.String("root", rootDir(), "root directory")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := requireEnv("RUBY_BOT_TOKEN")
	rubyChannelID := os.Getenv("RUBY_CHANNEL_ID")

	bot, err := discord.NewBot(token, rubyChannelID)
	if err != nil {
		slog.Error("creating bot", "err", err)
		os.Exit(1)
	}
	defer bot.Close()

	bot.Session.AddHandler(onReady())
	bot.Session.AddHandler(onMessageCreate(bot))

	if err := bot.Open(); err != nil {
		slog.Error("opening session", "err", err)
		os.Exit(1)
	}

	slog.Info("bot running")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
}

func onReady() func(*discordgo.Session, *discordgo.Ready) {
	return func(s *discordgo.Session, r *discordgo.Ready) {
		slog.Info("bot connected", "user", r.User.Username)
	}
}

// onMessageCreate reacts when the bot is mentioned or "Ruby" appears in a message.
func onMessageCreate(bot *discord.Bot) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		mentioned := false
		for _, u := range m.Mentions {
			if u.ID == s.State.User.ID {
				mentioned = true
				break
			}
		}

		if !mentioned && !strings.Contains(strings.ToLower(m.Content), "ruby") {
			return
		}

		slog.Info("bot triggered", "channel", m.ChannelID, "user", m.Author.Username, "content", m.Content)
		bot.Reply(m.ChannelID, m.ID, "Hey! You called?")
	}
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
