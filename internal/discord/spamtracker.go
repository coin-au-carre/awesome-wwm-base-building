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
	spamWindow        = 30 * time.Second
	spamChannelLimit  = 3               // distinct channels within the window to trigger
	spamCooldown      = 5 * time.Minute // min time between alerts for the same user
)

type spamEntry struct {
	channelID string
	at        time.Time
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

		// Append and prune old entries.
		entries := append(t.history[uid], spamEntry{m.ChannelID, now})
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
		msg := fmt.Sprintf("⚠️ **%s** (`%s`) posted in %d channels within 30s: %s",
			name, m.Author.Username, distinct, strings.Join(channels, ", "))

		bot.Send(t.alertCh, msg)
		slog.Info("spam alert", "user", m.Author.Username, "distinct_channels", distinct)
	}
}
