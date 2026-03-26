package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"ruby/internal/discord"
	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	root := flag.String("root", rootDir(), "root directory containing guilds.json")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := requireEnv("RUBY_BOT_TOKEN")
	botChannelID := os.Getenv("BOT_CHANNEL_ID")
	forumChannelID := requireEnv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")

	bot, err := discord.NewBot(token, botChannelID)
	if err != nil {
		slog.Error("creating bot", "err", err)
		os.Exit(1)
	}
	defer bot.Close()

	bot.Session.AddHandler(onReady())
	bot.Session.AddHandler(onThreadCreate(bot, *root, forumChannelID))
	bot.Session.AddHandler(onReactionAdd(bot, *root, forumChannelID))

	if err := bot.Open(); err != nil {
		slog.Error("opening session", "err", err)
		os.Exit(1)
	}

	slog.Info("bot running", "forum_channel", forumChannelID)
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

// onThreadCreate fires when a new post appears in the guild showcase forum.
// It triggers an incremental sync for that single thread.
func onThreadCreate(bot *discord.Bot, root, forumChannelID string) func(*discordgo.Session, *discordgo.ThreadCreate) {
	return func(s *discordgo.Session, t *discordgo.ThreadCreate) {
		if t.ParentID != forumChannelID {
			return
		}
		slog.Info("new forum thread", "name", t.Name, "id", t.ID)

		guilds, err := guild.Load(root)
		if err != nil {
			slog.Warn("loading guilds for thread sync", "err", err)
			guilds = []guild.Guild{}
		}

		updated, stats, err := discord.Sync(bot, guilds, discord.SyncConfig{
			ForumChannelID: forumChannelID,
		})
		if err != nil {
			slog.Error("thread sync failed", "thread", t.Name, "err", err)
			bot.Notify(fmt.Sprintf("💥 Failed to sync new thread **%s**: %s", t.Name, err))
			return
		}

		if err := guild.Save(root, updated); err != nil {
			slog.Error("saving guilds after thread sync", "err", err)
			return
		}

		bot.Notify(discord.FormatSyncSummary(stats))
		slog.Info("thread sync complete", "thread", t.Name)
	}
}

// onReactionAdd fires when a reaction is added to any message.
// Reactions on the first post of a showcase thread update that guild's score.
func onReactionAdd(bot *discord.Bot, root, forumChannelID string) func(*discordgo.Session, *discordgo.MessageReactionAdd) {
	return func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		ch, err := s.State.Channel(r.ChannelID)
		if err != nil || ch.ParentID != forumChannelID {
			return
		}
		slog.Info("reaction on showcase thread",
			"emoji", r.Emoji.Name,
			"channel", r.ChannelID,
			"user", r.UserID,
		)
		// TODO: targeted score update without a full sync
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
