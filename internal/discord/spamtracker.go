package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	spamWindow       = 10 * time.Second
	spamChannelLimit = 3               // distinct channels within the window to trigger
	spamCooldown     = 5 * time.Minute // min time between alerts for the same user
	spamTimeoutDur   = 24 * time.Hour
	spamWarnTimeout  = 1 * time.Hour
)

type spamEntry struct {
	channelID    string
	messageID    string
	content      string
	attachments  []string // CDN URLs
	fingerprints []string // filename:size, stable across re-uploads (unlike the CDN URL)
	at           time.Time
}

// SpamTracker flags users who post in many distinct channels in a short window.
type SpamTracker struct {
	mu      sync.Mutex
	history map[string][]spamEntry // userID → recent posts
	alerted map[string]time.Time   // userID → last alert time
	alertCh string
}

func NewSpamTracker(alertChannelID string) *SpamTracker {
	return &SpamTracker{
		history: make(map[string][]spamEntry),
		alerted: make(map[string]time.Time),
		alertCh: alertChannelID,
	}
}

func (t *SpamTracker) HandleMessage(bot *Bot) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if t.alertCh == "" || m.Author == nil || m.Author.Bot || m.GuildID == "" {
			return
		}

		now := time.Now()
		uid := m.Author.ID

		t.mu.Lock()

		urls := make([]string, len(m.Attachments))
		fingerprints := make([]string, len(m.Attachments))
		for i, a := range m.Attachments {
			urls[i] = a.URL
			fingerprints[i] = fmt.Sprintf("%s:%d", a.Filename, a.Size)
		}
		entries := append(t.history[uid], spamEntry{m.ChannelID, m.ID, m.Content, urls, fingerprints, now})
		cutoff := now.Add(-spamWindow)
		start := 0
		for start < len(entries) && entries[start].at.Before(cutoff) {
			start++
		}
		entries = entries[start:]
		t.history[uid] = entries

		// Count distinct channels.
		seen := make(map[string]bool, len(entries))
		for _, e := range entries {
			seen[e.channelID] = true
		}
		distinct := len(seen)

		// Check cool-down.
		lastAlert := t.alerted[uid]
		shouldAlert := distinct >= spamChannelLimit && now.Sub(lastAlert) >= spamCooldown
		if shouldAlert {
			t.alerted[uid] = now
		}

		// Snapshot entries for action outside the lock.
		snapshot := make([]spamEntry, len(entries))
		copy(snapshot, entries)

		t.mu.Unlock()

		if !shouldAlert {
			return
		}

		// Build channel list for the alert.
		channels := make([]string, 0, len(seen))
		for ch := range seen {
			channels = append(channels, "<#"+ch+">")
		}

		name := m.Author.GlobalName
		if name == "" {
			name = m.Author.Username
		}

		// Check if all messages in the window are identical (text + attachments).
		first := snapshot[0]
		isIdentical := true
		for _, e := range snapshot[1:] {
			if e.content != first.content || strings.Join(e.fingerprints, ",") != strings.Join(first.fingerprints, ",") {
				isIdentical = false
				break
			}
		}
		hasContent := first.content != "" || len(first.attachments) > 0

		if isIdentical && hasContent {
			// Timeout the user for 24h.
			until := now.Add(spamTimeoutDur)
			if err := s.GuildMemberTimeout(m.GuildID, uid, &until); err != nil {
				slog.Warn("spam timeout failed", "user", m.Author.Username, "err", err)
			}
			preview := first.content
			if len(preview) > 200 {
				preview = preview[:200] + "…"
			}
			msg := fmt.Sprintf(
				"🚫 **%s** (`%s`) timed out 24h for identical spam in %d channels within 20s: %s\n> %s",
				name, m.Author.Username, distinct, strings.Join(channels, ", "), preview,
			)
			if len(first.attachments) > 0 {
				msg += "\n" + strings.Join(first.attachments, "\n")
			}
			// Post evidence to the mod channel before deleting, since attachment
			// CDN links stop resolving once the source message is gone.
			bot.Send(t.alertCh, msg)
			for _, e := range snapshot {
				if err := s.ChannelMessageDelete(e.channelID, e.messageID); err != nil {
					slog.Warn("spam delete failed", "channel", e.channelID, "msg", e.messageID, "err", err)
				}
			}
			slog.Info("spam action: timeout + delete", "user", m.Author.Username, "distinct_channels", distinct)
		} else {
			until := now.Add(spamWarnTimeout)
			if err := s.GuildMemberTimeout(m.GuildID, uid, &until); err != nil {
				slog.Warn("spam timeout failed", "user", m.Author.Username, "err", err)
			}
			bot.Send(t.alertCh, fmt.Sprintf(
				"⚠️ **%s** (`%s`) silenced 1h for posting in %d channels within 20s: %s",
				name, m.Author.Username, distinct, strings.Join(channels, ", "),
			))
			for _, e := range snapshot {
				if err := s.ChannelMessageDelete(e.channelID, e.messageID); err != nil {
					slog.Warn("spam delete failed", "channel", e.channelID, "msg", e.messageID, "err", err)
				}
			}
			slog.Info("spam alert: timeout 1h + delete", "user", m.Author.Username, "distinct_channels", distinct)
		}
	}
}
