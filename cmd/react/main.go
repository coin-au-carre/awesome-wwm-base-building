// cmd/react/main.go
package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"ruby/internal/cmdutil"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var reactEmojis = []string{"👍", "🔥", "❤️", "⭐"}

func main() {
	solo := flag.Bool("solo", false, "react to SOLO_BUILD_SHOWCASE_CHANNEL_FORUM_ID instead of GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")
	dryRun := flag.Bool("dry-run", false, "fetch threads but skip adding reactions")
	name := flag.String("name", "", "only react to the thread whose name contains this substring (case-insensitive)")
	root := flag.String("root", cmdutil.RootDir(), "root directory containing .env")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	var forumID string
	if *solo {
		forumID = cmdutil.RequireEnv("SOLO_BUILD_SHOWCASE_CHANNEL_FORUM_ID")
	} else {
		forumID = cmdutil.RequireEnv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating session", "err", err)
		os.Exit(1)
	}
	s.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	if err := s.Open(); err != nil {
		slog.Error("opening session", "err", err)
		os.Exit(1)
	}
	defer s.Close()

	forumCh, err := s.Channel(forumID)
	if err != nil {
		slog.Error("fetching forum channel", "err", err)
		os.Exit(1)
	}
	guildID := forumCh.GuildID

	slog.Info("collecting threads...")
	threads, err := collectAllThreads(s, forumID, guildID)
	if err != nil {
		slog.Error("collecting threads", "err", err)
		os.Exit(1)
	}
	slog.Info("threads collected", "count", len(threads))

	if *name != "" {
		var filtered []*discordgo.Channel
		for _, t := range threads {
			if strings.Contains(strings.ToLower(t.Name), strings.ToLower(*name)) {
				filtered = append(filtered, t)
			}
		}
		slog.Info("filtered threads", "name", *name, "count", len(filtered))
		threads = filtered
	}

	f := false
	t2 := true

	added, skipped, failed := 0, 0, 0
	for i, t := range threads {
		slog.Info("processing thread", "n", i+1, "total", len(threads), "name", t.Name)
		// In Discord forum posts, the first message ID equals the thread (channel) ID.
		msgID := t.ID

		isArchived := t.ThreadMetadata != nil && t.ThreadMetadata.Archived
		if isArchived && !*dryRun {
			if _, err := s.ChannelEditComplex(t.ID, &discordgo.ChannelEdit{Archived: &f, Locked: &f}); err != nil {
				slog.Warn("failed to unarchive thread, skipping", "thread", t.Name, "err", err)
				failed += len(reactEmojis)
				continue
			}
		}

		threadAdded := 0
		for _, emoji := range reactEmojis {
			if *dryRun {
				slog.Info("dry-run: would react", "emoji", emoji)
				skipped++
				continue
			}
			if err := s.MessageReactionAdd(t.ID, msgID, emoji); err != nil {
				// 90001 = already reacted, not a real failure
				if restErr, ok := err.(*discordgo.RESTError); ok && restErr.Message != nil && restErr.Message.Code == 90001 {
					skipped++
				} else {
					slog.Warn("reaction failed", "emoji", emoji, "err", err)
					failed++
				}
			} else {
				added++
				threadAdded++
			}
			// Respect Discord rate limits.
			time.Sleep(250 * time.Millisecond)
		}

		if isArchived && !*dryRun {
			if _, err := s.ChannelEditComplex(t.ID, &discordgo.ChannelEdit{Archived: &t2}); err != nil {
				slog.Warn("failed to re-archive thread", "thread", t.Name, "err", err)
			}
		}

		if threadAdded > 0 {
			slog.Info("reacted", "thread", t.Name, "new", threadAdded)
		}
	}

	slog.Info("done", "added", added, "already_present", skipped, "failed", failed)
}

func collectAllThreads(s *discordgo.Session, forumID, guildID string) ([]*discordgo.Channel, error) {
	active, err := s.GuildThreadsActive(guildID)
	if err != nil {
		return nil, err
	}

	var threads []*discordgo.Channel
	for _, t := range active.Threads {
		if t.ParentID == forumID {
			threads = append(threads, t)
		}
	}

	// Paginate through archived threads.
	var before *time.Time
	page := 1
	for {
		archived, err := s.ThreadsArchived(forumID, before, 100)
		if err != nil {
			return nil, err
		}
		threads = append(threads, archived.Threads...)
		slog.Info("archived threads page", "page", page, "fetched", len(archived.Threads), "total_so_far", len(threads))
		page++
		if !archived.HasMore || len(archived.Threads) == 0 {
			break
		}
		ts := archived.Threads[len(archived.Threads)-1].ThreadMetadata.ArchiveTimestamp
		before = &ts
	}

	return threads, nil
}
