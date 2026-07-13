// internal/discord/instancelock.go
package discord

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ponytail: lock lives in a single Discord message (edited as a heartbeat)
// instead of a shared DB/Redis — local and VPS share no filesystem, but both
// already talk to Discord, so no new infra is needed.
const (
	// DefaultInstanceLockMessageID is the fallback used when
	// INSTANCE_LOCK_MESSAGE_ID isn't set, so every instance locks against the
	// same message instead of each silently creating its own (and never
	// seeing each other) the first time it runs without that env var set.
	DefaultInstanceLockMessageID = "1526335789347246251"

	lockHeartbeatInterval = 5 * time.Second
	lockStaleAfter        = 3 * lockHeartbeatInterval // ~45s: 3 missed beats before a waiter takes over
)

// AcquireLock blocks until it can safely claim the lock (no fresher, equal-
// or-higher-priority holder), then starts a background goroutine that
// re-writes the lock message every lockHeartbeatInterval. If a higher
// priority instance later takes the lock away, that goroutine calls cancel
// so this process shuts down instead of fighting the new holder for events.
// Call before bot.Open() so a waiting instance never opens a gateway
// connection.
func AcquireLock(ctx context.Context, cancel context.CancelFunc, s *discordgo.Session, channelID, messageID, instanceID string, priority int) {
	if channelID == "" {
		slog.Warn("no lock channel configured (DEV_CHANNEL_ID), skipping instance lock")
		return
	}

	for messageID != "" {
		holder, holderPriority, ts, ok := readLock(s, channelID, messageID)
		if !ok || holder == instanceID || time.Since(ts) > lockStaleAfter || priority > holderPriority {
			break
		}
		slog.Info("another bot instance holds the lock, waiting", "holder", holder, "priority", holderPriority, "age", time.Since(ts).Round(time.Second))
		select {
		case <-ctx.Done():
			return
		case <-time.After(lockHeartbeatInterval):
		}
	}

	messageID = writeLock(s, channelID, messageID, instanceID, priority)
	slog.Info("acquired instance lock", "instance", instanceID, "priority", priority)

	go func() {
		t := time.NewTicker(lockHeartbeatInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if holder, _, _, ok := readLock(s, channelID, messageID); ok && holder != instanceID {
					slog.Info("lock taken over by another instance, shutting down", "new_holder", holder)
					cancel()
					return
				}
				writeLock(s, channelID, messageID, instanceID, priority)
			}
		}
	}()
}

// lockContentRe parses messages written by lockContent: holder, priority, heartbeat unix time.
var lockContentRe = regexp.MustCompile("lock held by `([^`]+)` \\(priority (-?\\d+)\\) — heartbeat (\\d+)")

func lockContent(instanceID string, priority int, ts time.Time) string {
	return fmt.Sprintf("🔒 lock held by `%s` (priority %d) — heartbeat %d", instanceID, priority, ts.Unix())
}

// parseLockContent extracts the holder name, priority, and heartbeat time
// from a lock message body previously written by lockContent.
func parseLockContent(content string) (holder string, priority int, ts time.Time, ok bool) {
	m := lockContentRe.FindStringSubmatch(content)
	if m == nil {
		return "", 0, time.Time{}, false
	}
	priority, err := strconv.Atoi(m[2])
	if err != nil {
		return "", 0, time.Time{}, false
	}
	unix, err := strconv.ParseInt(m[3], 10, 64)
	if err != nil {
		return "", 0, time.Time{}, false
	}
	return m[1], priority, time.Unix(unix, 0), true
}

func readLock(s *discordgo.Session, channelID, messageID string) (holder string, priority int, ts time.Time, ok bool) {
	m, err := s.ChannelMessage(channelID, messageID)
	if err != nil {
		slog.Warn("reading lock message", "err", err)
		return "", 0, time.Time{}, false
	}
	return parseLockContent(m.Content)
}

// writeLock edits messageID in place, or creates it on the first run when
// empty, returning the message ID to reuse on subsequent heartbeats.
func writeLock(s *discordgo.Session, channelID, messageID, instanceID string, priority int) string {
	content := lockContent(instanceID, priority, time.Now())
	if messageID != "" {
		if _, err := s.ChannelMessageEdit(channelID, messageID, content); err == nil {
			return messageID
		}
	}
	msg, err := s.ChannelMessageSend(channelID, content)
	if err != nil {
		slog.Warn("writing lock message", "err", err)
		return messageID
	}
	slog.Info("created new lock message — set INSTANCE_LOCK_MESSAGE_ID env var", "messageID", msg.ID)
	return msg.ID
}
