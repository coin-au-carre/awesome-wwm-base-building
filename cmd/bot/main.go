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
	noClaude := flag.Bool("no-claude", false, "disable Claude responses (slash commands still work)")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	activeChannelID := cmdutil.RequireEnv("RUBY_CHANNEL_ID")
	allowedChannels := map[string]bool{activeChannelID: true}
	if devChannelID := os.Getenv("DEV_CHANNEL_ID"); devChannelID != "" {
		allowedChannels[devChannelID] = true
	}
	slog.Info("bot mode", "channels", len(allowedChannels))

	guildForumID := os.Getenv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")
	soloForumID := os.Getenv("SOLO_BUILD_SHOWCASE_CHANNEL_FORUM_ID")

	bot, err := discord.NewBot(token, activeChannelID)
	if err != nil {
		slog.Error("creating bot", "err", err)
		os.Exit(1)
	}
	defer bot.Close()

	var responder *discord.Responder
	if !*noClaude {
		responder = buildResponder(*root)
	}

	discordGuildID := os.Getenv("DISCORD_GUILD_ID")

	rubyRoleID := os.Getenv("RUBY_ROLE_ID")
	submissionChannelID := os.Getenv("GUILD_SUBMISSION_CHANNEL_ID")
	discoveriesChannelID := os.Getenv("GUILD_DISCOVERIES_CHANNEL_ID")
	bot.Session.AddHandler(onReady(discordGuildID, discoveriesChannelID))
	bot.Session.AddHandler(onMessageCreate(bot, responder, *root, allowedChannels, rubyRoleID))
	bot.Session.AddHandler(discord.OnInteractionCreate(bot, *root, submissionChannelID, discoveriesChannelID, guildForumID, soloForumID, responder))
	bot.Session.AddHandler(onGuildMemberAdd())

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

func onReady(discordGuildID, discoveriesChannelID string) func(*discordgo.Session, *discordgo.Ready) {
	return func(s *discordgo.Session, r *discordgo.Ready) {
		slog.Info("bot connected", "user", r.User.Username)
		discord.RegisterSubmitCommand(s, discordGuildID)
		logRegisteredCommands(s, discordGuildID)
		logChannelPermissions(s, r.User.ID, discoveriesChannelID)
	}
}

func logRegisteredCommands(s *discordgo.Session, discordGuildID string) {
	cmds, err := s.ApplicationCommands(s.State.User.ID, discordGuildID)
	if err != nil {
		slog.Warn("could not list registered commands", "err", err)
		return
	}
	names := make([]string, 0, len(cmds))
	for _, c := range cmds {
		names = append(names, "/"+c.Name)
	}
	slog.Info("registered commands", "commands", names)
}

func logChannelPermissions(s *discordgo.Session, botUserID, channelID string) {
	if channelID == "" {
		slog.Warn("channel permission check skipped: channel ID not set")
		return
	}
	perms, err := s.UserChannelPermissions(botUserID, channelID)
	if err != nil {
		slog.Warn("could not check channel permissions", "channel", channelID, "err", err)
		return
	}
	canView := perms&discordgo.PermissionViewChannel != 0
	canSend := perms&discordgo.PermissionSendMessages != 0
	slog.Info("channel permissions", "channel", channelID, "view", canView, "send", canSend)
}

var reMention = regexp.MustCompile(`<@[!&]?\d+>`)

// spotlightKeywords are exact single-word triggers that bypass Claude entirely.
var spotlightKeywords = map[string]bool{
	"random": true,
}

// onMessageCreate reacts when the bot is mentioned or "Ruby" appears in a message.
func onMessageCreate(bot *discord.Bot, responder *discord.Responder, root string, allowedChannels map[string]bool, rubyRoleID string) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		// Log DMs so we know someone tried to reach Ruby privately.
		if m.GuildID == "" {
			slog.Info("dm received", "username", m.Author.Username, "display_name", m.Author.GlobalName, "user_id", m.Author.ID, "content", m.Content)
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

		roleMentioned := rubyRoleID != "" && strings.Contains(m.Content, "<@&"+rubyRoleID+">")
		if !mentioned && !roleMentioned && !strings.Contains(strings.ToLower(m.Content), "ruby") {
			return
		}

		// Strip mention tags and trim so we get clean text.
		text := strings.TrimSpace(reMention.ReplaceAllString(m.Content, ""))
		if text == "" {
			text = "Hello!"
		}

		slog.Info("bot triggered", "channel", m.ChannelID, "user", m.Author.Username, "content", text)

		// Fast path: single keyword commands skip Claude entirely.
		for _, word := range strings.Fields(strings.ToLower(text)) {
			if spotlightKeywords[word] {
				handleSpotlightReply(bot, s, responder, m.ChannelID, m.ID, root)
				return
			}
		}

		if responder == nil {
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

		if result.ShowSolo {
			handleSoloSpotlightReply(bot, s, responder, m.ChannelID, m.ID, root)
			return
		}

		if result.GuildImageQuery != "" {
			handleGuildImageReply(bot, s, responder, m.ChannelID, m.ID, root, result.GuildImageQuery)
			return
		}

		if result.CatalogQuery != "" {
			handleCatalogItemsReply(bot, s, m.ChannelID, m.ID, root, result.CatalogQuery)
			return
		}

		bot.Reply(m.ChannelID, m.ID, result.Text)
	}
}

func onGuildMemberAdd() func(*discordgo.Session, *discordgo.GuildMemberAdd) {
	return func(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
		name := m.User.GlobalName
		if name == "" {
			name = m.User.Username
		}
		slog.Info("sending welcome message", "user", m.User.Username, "display_name", name)
		ch, err := s.UserChannelCreate(m.User.ID)
		if err != nil {
			slog.Warn("failed to open DM channel for welcome", "user", m.User.Username, "err", err)
			return
		}
		if _, err := s.ChannelMessageSend(ch.ID, discord.BuildWelcomeMessage(name)); err != nil {
			slog.Warn("failed to send welcome DM", "user", m.User.Username, "err", err)
		}
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
