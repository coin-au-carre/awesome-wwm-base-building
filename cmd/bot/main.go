package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	root := flag.String("root", cmdutil.RootDir(), "root directory")
	dev := flag.Bool("dev", false, "use DEV_CHANNEL_ID instead of RUBY_CHANNEL_ID")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	channelEnvKey := "RUBY_CHANNEL_ID"
	if *dev {
		channelEnvKey = "DEV_CHANNEL_ID"
	}
	activeChannelID := cmdutil.RequireEnv(channelEnvKey)
	slog.Info("bot mode", "channel_env", channelEnvKey, "channel", activeChannelID)

	allowedChannels := map[string]bool{activeChannelID: true}

	bot, err := discord.NewBot(token, activeChannelID)
	if err != nil {
		slog.Error("creating bot", "err", err)
		os.Exit(1)
	}
	defer bot.Close()

	responder := buildResponder(*root)

	bot.Session.AddHandler(onReady())
	bot.Session.AddHandler(onMessageCreate(bot, responder, *root, allowedChannels))

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

var reMention = regexp.MustCompile(`<@!?\d+>`)

// spotlightKeywords are exact single-word triggers that bypass Claude entirely.
var spotlightKeywords = map[string]bool{
	"spotlight": true,
	"random":    true,
}

// onMessageCreate reacts when the bot is mentioned or "Ruby" appears in a message.
func onMessageCreate(bot *discord.Bot, responder *discord.Responder, root string, allowedChannels map[string]bool) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		// Ignore messages outside allowed channels.
		if !allowedChannels[m.ChannelID] {
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

		// Strip mention tags and trim so we get clean text.
		text := strings.TrimSpace(reMention.ReplaceAllString(m.Content, ""))
		if text == "" {
			text = "Hello!"
		}

		slog.Info("bot triggered", "channel", m.ChannelID, "user", m.Author.Username, "content", text)

		// Fast path: single keyword commands skip Claude entirely.
		if spotlightKeywords[strings.ToLower(text)] {
			handleSpotlightReply(bot, s, responder, m.ChannelID, m.ID, root)
			return
		}

		// Collect image attachment URLs.
		var imageURLs []string
		for _, a := range m.Attachments {
			if strings.HasPrefix(a.ContentType, "image/") {
				imageURLs = append(imageURLs, a.URL)
			}
		}

		_ = s.ChannelTyping(m.ChannelID)

		result, err := responder.Reply(context.Background(), m.ChannelID, text, imageURLs)
		if err != nil {
			slog.Error("claude reply", "err", err)
			bot.Reply(m.ChannelID, m.ID, "*(the winds are silent for now… try again in a moment.)*")
			return
		}

		if result.ShowSpotlight {
			handleSpotlightReply(bot, s, responder, m.ChannelID, m.ID, root)
			return
		}

		if result.GuildImageQuery != "" {
			handleGuildImageReply(bot, s, responder, m.ChannelID, m.ID, root, result.GuildImageQuery)
			return
		}

		bot.Reply(m.ChannelID, m.ID, result.Text)
	}
}

// buildResponder returns a Responder using ANTHROPIC_API_KEY if set,
// otherwise falls back to the `claude` CLI (Pro subscription via Claude Code).
func buildResponder(root string) *discord.Responder {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		slog.Info("claude: using ANTHROPIC_API_KEY")
		c := anthropic.NewClient(option.WithAPIKey(key))
		return discord.NewResponder(&c, root)
	}
	slog.Info("claude: no API key found, using Claude Code CLI (Pro subscription)")
	return discord.NewCLIResponder(root)
}
