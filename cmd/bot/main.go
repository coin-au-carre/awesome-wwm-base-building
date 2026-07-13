package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/bwmarrin/discordgo"
)

func main() {
	root := flag.String("root", cmdutil.RootDir(), "root directory")
	noClaude := flag.Bool("no-claude", false, "disable Claude responses (slash commands still work)")
	welcomeDM := flag.Bool("welcome-dm", false, "enable welcome DM sent to new members")
	useOllama := flag.Bool("ollama", false, "use local Ollama instead of Claude")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama API endpoint")
	ollamaModel := flag.String("ollama-model", "llama3.1", "Ollama model name")
	flag.Parse()

	cmdutil.LoadEnv(*root)

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	activeChannelID := cmdutil.RequireEnv("RUBY_CHANNEL_ID")
	devChannelID := os.Getenv("DEV_CHANNEL_ID")
	generalChannelID := os.Getenv("GENERAL_CHANNEL_ID")
	allowedChannels := map[string]bool{activeChannelID: true}
	if devChannelID != "" {
		allowedChannels[devChannelID] = true
	}
	spotlightOnlyChannels := map[string]bool{}
	if generalChannelID != "" {
		spotlightOnlyChannels[generalChannelID] = true
	}
	slog.Info("bot mode", "channels", len(allowedChannels), "spotlight_only_channels", len(spotlightOnlyChannels))

	guildForumID := os.Getenv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")
	soloForumID := os.Getenv("SOLO_BUILD_SHOWCASE_CHANNEL_FORUM_ID")

	bot, err := discord.NewBot(token, activeChannelID)
	if err != nil {
		slog.Error("creating bot", "err", err)
		os.Exit(1)
	}
	defer bot.Close()

	var responder discord.LLMResponder
	if !*noClaude {
		responder = buildResponder(*root, *useOllama, *ollamaURL, *ollamaModel)
	}

	discordGuildID := os.Getenv("DISCORD_GUILD_ID")

	rubyRoleID := os.Getenv("RUBY_ROLE_ID")
	submissionChannelID := os.Getenv("GUILD_SUBMISSION_CHANNEL_ID")
	discoveriesChannelID := os.Getenv("GUILD_DISCOVERIES_CHANNEL_ID")
	logsChannelID := os.Getenv("LOGS_CHANNEL_ID")
	moderationChannelID := os.Getenv("MODERATION_CHANNEL_ID")
	if moderationChannelID == "" {
		moderationChannelID = logsChannelID
	}
	botChannelID := os.Getenv("BOT_CHANNEL_ID")
	trustedEyeRoleID := os.Getenv("TRUSTED_EYE_ROLE_ID")
	trustedMemberRoleID := os.Getenv("TRUSTED_MEMBER_ROLE_ID")
	githubToken := os.Getenv("GITHUB_ACTIONS_TOKEN")
	streamingTracker := discord.NewStreamingTracker(*root, bot.Session, discordGuildID)
	spamTracker := discord.NewSpamTracker(moderationChannelID)
	inviteTracker := discord.NewInviteTracker(discordGuildID, logsChannelID)
	bot.Session.AddHandler(onReady(discordGuildID))
	bot.Session.AddHandler(inviteTracker.OnReady)
	bot.Session.AddHandler(streamingTracker.HandleGuildCreate)
	bot.Session.AddHandler(onMessageCreate(bot, responder, *root, allowedChannels, spotlightOnlyChannels, activeChannelID, rubyRoleID))
	bot.Session.AddHandler(spamTracker.HandleMessage(bot))
	bot.Session.AddHandler(discord.OnInteractionCreate(bot, *root, submissionChannelID, discoveriesChannelID, guildForumID, soloForumID, devChannelID, botChannelID, trustedEyeRoleID, trustedMemberRoleID, githubToken, responder))
	bot.Session.AddHandler(inviteTracker.OnMemberAdd(bot))
	bot.Session.AddHandler(inviteTracker.OnInviteCreate)
	bot.Session.AddHandler(inviteTracker.OnInviteDelete)
	if *welcomeDM {
		bot.Session.AddHandler(onGuildMemberAdd())
	}
	bot.Session.AddHandler(onGuildMemberRemove(bot, logsChannelID))
	bot.Session.AddHandler(onGuildMemberUpdate(bot, moderationChannelID))
	bot.Session.AddHandler(onHomesteadRoleUpdate(bot, *root, devChannelID))
	bot.Session.AddHandler(onHoneypotChannelPost(bot, moderationChannelID))
	bot.Session.AddHandler(streamingTracker.HandleVoiceStateUpdate)
	bot.Session.AddHandler(discord.HandleHexiPartyMute)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	instanceID := os.Getenv("INSTANCE_NAME")
	if instanceID == "" {
		if h, err := os.Hostname(); err == nil {
			instanceID = h
		} else {
			instanceID = "unknown"
		}
	}
	lockMessageID := os.Getenv("INSTANCE_LOCK_MESSAGE_ID")
	if lockMessageID == "" {
		lockMessageID = discord.DefaultInstanceLockMessageID
	}
	instancePriority, err := strconv.Atoi(os.Getenv("INSTANCE_PRIORITY"))
	if err != nil {
		instancePriority = 0
	}

	onAcquire := func(activeCtx context.Context) {
		if err := bot.Open(); err != nil {
			slog.Error("opening session", "err", err)
			return
		}
		discord.PullOnStart(*root, responder)
		discord.StartDataWatcher(activeCtx, *root, responder)
		slog.Info("bot running")
	}
	onRelease := func() {
		bot.Close()
	}
	discord.RunLocked(ctx, bot.Session, devChannelID, lockMessageID, instanceID, instancePriority, onAcquire, onRelease)

	slog.Info("shutting down")
}

func onReady(discordGuildID string) func(*discordgo.Session, *discordgo.Ready) {
	return func(s *discordgo.Session, r *discordgo.Ready) {
		guilds := make([]string, 0, len(r.Guilds))
		for _, g := range r.Guilds {
			name := g.ID
			if full, err := s.Guild(g.ID); err == nil {
				name = full.Name + " (" + g.ID + ")"
			}
			guilds = append(guilds, name)
		}
		slog.Info("bot connected", "user", r.User.Username, "guilds", guilds)
		discord.RegisterSubmitCommand(s, discordGuildID)
		logRegisteredCommands(s, discordGuildID)
	}
}

func logRegisteredCommands(s *discordgo.Session, discordGuildID string) {
	globalCmds, err := s.ApplicationCommands(s.State.User.ID, "")
	if err != nil {
		slog.Warn("could not list global commands", "err", err)
	} else {
		names := make([]string, 0, len(globalCmds))
		for _, c := range globalCmds {
			names = append(names, "/"+c.Name)
		}
		slog.Info("registered global commands", "commands", names)
	}

	guildCmds, err := s.ApplicationCommands(s.State.User.ID, discordGuildID)
	if err != nil {
		slog.Warn("could not list guild commands", "err", err)
		return
	}
	names := make([]string, 0, len(guildCmds))
	for _, c := range guildCmds {
		names = append(names, "/"+c.Name)
	}
	slog.Info("registered guild commands", "commands", names)
}

var reMention = regexp.MustCompile(`<@[!&]?\d+>`)

// spotlightKeywords are exact single-word triggers that bypass Claude entirely.
var spotlightKeywords = map[string]bool{
	"random": true,
}

// faqAnswers are exact canned replies for common questions, matched by required keyword sets, bypassing Claude entirely.
var faqAnswers = []struct {
	keywords []string
	answer   string
}{
	{[]string{"submit", "solo"}, "Use `/submit-solo` — I'll DM you a ready-to-paste template. Your build appears on the next sync! Check the contribute guide too: https://www.wherebuildersmeet.com/contribute/builder"},
	{[]string{"submit", "guild"}, "Use `/submit-guild` — I'll DM you a ready-to-paste template. Your build appears on the next sync! Check the contribute guide too: https://www.wherebuildersmeet.com/contribute/builder"},
	{[]string{"scout"}, "Use `/scout-guild` to report an impressive base you've found!"},
	{[]string{"find", "builder"}, "Find your match in <#1513104617041297498>!"},
	{[]string{"looking for", "builder"}, "Find your match in <#1513104617041297498>!"},
	{[]string{"blueprint"}, "Ask the builders in <#1502979217619685416>!"},
	{[]string{"tutorial"}, "Browse tutorials at https://www.wherebuildersmeet.com/tutorials/ or ask in <#1483483683456286911> or <#1522001435187744901>!"},
	{[]string{"vote"}, "Read the voting guide at https://www.wherebuildersmeet.com/how-it-works/ before you start — explore many bases and rate each one honestly!"},
}

// liteModeNags tracks repeated triggers per channel+user in lite-mode channels
// so Ruby can get annoyed and redirect pests to #ruby instead of answering forever.
var liteModeNags = struct {
	mu   sync.Mutex
	seen map[string]int
	last map[string]time.Time
}{seen: make(map[string]int), last: make(map[string]time.Time)}

const liteModeNagWindow = 5 * time.Minute
const liteModeNagThreshold = 3

// annoyedResponses are canned replies once someone trips the lite-mode nag threshold.
var annoyedResponses = []string{
	"*(sighs)* that's a lot of questions for one chat... take it to <#%s>?",
	"okay, i think this deserves its own room. <#%s> is right there~",
	"*(fans self)* you're keeping me busy! let's continue in <#%s> instead.",
}

// checkLiteModeNag returns an annoyed reply and true if this channel+user has
// tripped the repeat-question threshold within the window, resetting the count.
func checkLiteModeNag(channelID, userID, rubyChannelID string) (string, bool) {
	key := channelID + ":" + userID
	now := time.Now()

	liteModeNags.mu.Lock()
	defer liteModeNags.mu.Unlock()

	if now.Sub(liteModeNags.last[key]) > liteModeNagWindow {
		liteModeNags.seen[key] = 0
	}
	liteModeNags.seen[key]++
	liteModeNags.last[key] = now

	if liteModeNags.seen[key] >= liteModeNagThreshold {
		liteModeNags.seen[key] = 0
		msg := annoyedResponses[rand.Intn(len(annoyedResponses))]
		return fmt.Sprintf(msg, rubyChannelID), true
	}
	return "", false
}

// restingResponses are random messages Ruby gives when Claude is disabled.
var restingResponses = []string{
	"*(rests quietly)* i'm taking a moment away~",
	"shhh... the winds are quiet right now...",
	"*(curls up)* resting... come back later?",
	"hmmm... can't quite think right now... 💤",
	"*(drifts off)* ...do not disturb...",
	"the world feels sleepy to me... try later?",
	"*(loses focus)* sorry, i'm not quite here...",
	"resting between moments... 💭",
}

// onMessageCreate reacts when the bot is mentioned or "Ruby" appears in a message.
func onMessageCreate(bot *discord.Bot, responder discord.LLMResponder, root string, allowedChannels map[string]bool, spotlightOnlyChannels map[string]bool, rubyChannelID, rubyRoleID string) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		// Log DMs so we know someone tried to reach Ruby privately.
		if m.GuildID == "" {
			slog.Info("dm received", "username", m.Author.Username, "display_name", m.Author.GlobalName, "user_id", m.Author.ID, "content", m.Content)
			return
		}

		// Determine channel mode.
		liteMode := !allowedChannels[m.ChannelID]

		// Use the literal mention tag in content, not m.Mentions — Discord auto-adds
		// the replied-to author to m.Mentions on any reply, even without an @ mention.
		mentioned := strings.Contains(m.Content, "<@"+s.State.User.ID+">") || strings.Contains(m.Content, "<@!"+s.State.User.ID+">")

		roleMentioned := rubyRoleID != "" && strings.Contains(m.Content, "<@&"+rubyRoleID+">")
		if !mentioned && !roleMentioned && !strings.Contains(strings.ToLower(m.Content), "ruby") {
			return
		}

		// Strip mention tags and trim so we get clean text.
		text := strings.TrimSpace(reMention.ReplaceAllString(m.Content, ""))
		if text == "" {
			text = "Hello!"
		}

		displayName := m.Author.Username
		if m.Member != nil && m.Member.Nick != "" {
			displayName = m.Member.Nick
		}
		slog.Info("bot triggered", "channel", m.ChannelID, "display_name", displayName, "content", text)

		// Fast path: single keyword commands skip Claude entirely.
		lowerText := strings.ToLower(text)
		for _, word := range strings.Fields(lowerText) {
			if spotlightKeywords[word] {
				handleSpotlightReply(bot, s, responder, m.ChannelID, m.ID, root)
				return
			}
		}

		// Fast path: common questions get a canned answer, skipping the LLM call entirely.
		for _, faq := range faqAnswers {
			matched := true
			for _, kw := range faq.keywords {
				if !strings.Contains(lowerText, kw) {
					matched = false
					break
				}
			}
			if matched {
				bot.Reply(m.ChannelID, m.ID, faq.answer)
				return
			}
		}

		// Outside #ruby/#dev: tutorial/technique questions only, short replies, redirect searches.
		if liteMode {
			if annoyed, ok := checkLiteModeNag(m.ChannelID, m.Author.ID, rubyChannelID); ok {
				bot.Reply(m.ChannelID, m.ID, annoyed)
				return
			}
			text = "[You are replying in a public channel outside #ruby. Answer only building technique or tutorial questions, in 1-2 sentences max. For guild searches, spotlights, or image requests, invite the user to ask in <#" + rubyChannelID + "> instead.]\n\n" + text
		}

		if responder == nil {
			// Pick a random resting response
			idx := rand.Intn(len(restingResponses))
			bot.Reply(m.ChannelID, m.ID, restingResponses[idx])
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

		if result.AllowEmbeds {
			bot.ReplyWithEmbeds(m.ChannelID, m.ID, result.Text)
		} else {
			bot.ReplyChunked(m.ChannelID, m.ID, result.Text)
		}
	}
}

func onGuildMemberRemove(bot *discord.Bot, logsChannelID string) func(*discordgo.Session, *discordgo.GuildMemberRemove) {
	return func(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
		if logsChannelID == "" {
			return
		}
		name := m.User.GlobalName
		if name == "" {
			name = m.User.Username
		}
		guildName := m.GuildID
		if g, err := s.Guild(m.GuildID); err == nil {
			guildName = fmt.Sprintf("%s (%s)", g.Name, m.GuildID)
		}
		msg := fmt.Sprintf("🥀 **%s** (`%s`) left **%s**.", name, m.User.Username, guildName)
		if m.Member != nil && !m.Member.JoinedAt.IsZero() {
			dur := time.Since(m.Member.JoinedAt)
			days := int(dur.Hours() / 24)
			switch {
			case days < 1:
				msg += " Joined today."
			case days == 1:
				msg += " Joined 1 day ago."
			default:
				msg += fmt.Sprintf(" Joined %d days ago.", days)
			}
		}
		bot.Send(logsChannelID, msg)
		slog.Info("member left", "user", m.User.Username, "display_name", name, "discord_server", guildName)
	}
}

// prisonnerRoleID is a honeypot role: any legitimate member gaining it signals
// a compromised/malicious bot or a moderator mis-click, so it's hardcoded
// rather than env-configured like the other roles.
const prisonnerRoleID = "1521804658216272033"

// maxTimeoutDur is Discord's hard ceiling on member timeouts.
const maxTimeoutDur = 28 * 24 * time.Hour

func onGuildMemberUpdate(bot *discord.Bot, moderationChannelID string) func(*discordgo.Session, *discordgo.GuildMemberUpdate) {
	return func(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
		if moderationChannelID == "" || m.Member == nil {
			return
		}
		hadBefore := m.BeforeUpdate != nil && slices.Contains(m.BeforeUpdate.Roles, prisonnerRoleID)
		hasNow := slices.Contains(m.Member.Roles, prisonnerRoleID)
		if hasNow && !hadBefore {
			name := m.User.GlobalName
			if name == "" {
				name = m.User.Username
			}
			until := time.Now().Add(maxTimeoutDur)
			if err := s.GuildMemberTimeout(m.GuildID, m.User.ID, &until); err != nil {
				slog.Warn("honeypot role timeout failed", "user", m.User.Username, "err", err)
			}
			bot.Send(moderationChannelID, fmt.Sprintf(
				"🍯 Honeypot triggered: **%s** (`%s`) was given the prisonner role and was silenced 28d.", name, m.User.Username,
			))
			slog.Info("honeypot role granted", "user", m.User.Username)
		}
	}
}

// onHomesteadRoleUpdate reacts to a member's Homestead level role changing
// in real time, instead of waiting for the hourly sync-homestead workflow to
// pick it up. The hourly Action stays in place as a backstop for role
// changes made while the bot is offline, and for the initial backfill.
func onHomesteadRoleUpdate(bot *discord.Bot, root, devChannelID string) func(*discordgo.Session, *discordgo.GuildMemberUpdate) {
	return func(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
		if m.Member == nil {
			return
		}
		newLevel := discord.HomesteadLevelFromRoles(m.Member.Roles)
		if newLevel == 0 {
			return
		}

		members, err := discord.LoadHomesteadMembers(root)
		if err != nil {
			slog.Error("loading homestead_members.json", "err", err)
			return
		}

		// Compare against the level on disk rather than discordgo's in-memory
		// BeforeUpdate: on a large guild Discord doesn't send a full member
		// list on connect, so BeforeUpdate is nil (and this check falls
		// through as a no-op level-up) until the bot happens to have already
		// cached that member from some earlier event.
		if members[m.User.ID].Level >= newLevel {
			return
		}

		entry := discord.HomesteadMember{
			Level:      newLevel,
			Since:      time.Now().UTC().Format("2006-01-02 15:04"),
			Username:   m.User.Username,
			GlobalName: m.User.GlobalName,
			Nickname:   m.Member.Nick,
		}
		members[m.User.ID] = entry

		if err := discord.SaveHomesteadMembers(root, members); err != nil {
			slog.Error("saving homestead_members.json", "err", err)
			return
		}
		go discord.GitCommitAndPush(root, "data/homestead_members.json", "data: homestead level up", bot, devChannelID)

		messageID := os.Getenv("HOMESTEAD_MESSAGE_ID")
		if messageID == "" {
			messageID = discord.DefaultHomesteadMessageID
		}
		discord.PostHomesteadRanking(s, messageID, members)

		if newLevel >= 7 {
			discord.AnnounceHomesteadLevelUp(s, members, entry, m.User.ID)
		}
		slog.Info("homestead level up", "user", m.User.Username, "level", newLevel)
	}
}

// honeypotChannelID is a hidden channel: no legitimate member should ever
// see or post in it, so posting there is treated as malicious. Hardcoded
// for the same reason as prisonnerRoleID.
const honeypotChannelID = "1521804936428392559"

func onHoneypotChannelPost(bot *discord.Bot, moderationChannelID string) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.ChannelID != honeypotChannelID || m.Author == nil || m.Author.Bot || m.GuildID == "" {
			return
		}
		name := m.Author.GlobalName
		if name == "" {
			name = m.Author.Username
		}
		if err := s.GuildBanCreateWithReason(m.GuildID, m.Author.ID, "honeypot channel post", 0); err != nil {
			slog.Warn("honeypot ban failed", "user", m.Author.Username, "err", err)
		}
		preview := m.Content
		if len(preview) > 200 {
			preview = preview[:200] + "…"
		}
		bot.Send(moderationChannelID, fmt.Sprintf(
			"🍯 Honeypot triggered: **%s** (`%s`) posted in the honeypot channel and was banned.\n> %s",
			name, m.Author.Username, preview,
		))
		slog.Info("honeypot channel post", "user", m.Author.Username)
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

// buildResponder returns a Responder using Ollama, Claude API, or Claude Code CLI.
func buildResponder(root string, useOllama bool, ollamaURL, ollamaModel string) discord.LLMResponder {
	if useOllama {
		slog.Info("using ollama", "url", ollamaURL, "model", ollamaModel)
		return discord.NewOllamaResponder(ollamaURL, ollamaModel, root)
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		slog.Info("claude: using ANTHROPIC_API_KEY")
		c := anthropic.NewClient(option.WithAPIKey(key))
		return discord.NewResponder(&c, root)
	}
	slog.Info("claude: no API key found, using Claude Code CLI (Pro subscription)")
	return discord.NewCLIResponder(root)
}
