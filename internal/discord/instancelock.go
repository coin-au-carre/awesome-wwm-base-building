// internal/discord/instancelock.go
package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ponytail: lock lives in a single Discord message (edited as a heartbeat)
// instead of a shared DB/Redis — local and VPS share no filesystem, but both
// already talk to Discord, so no new infra is needed.
const (
	lockHeartbeatInterval = 5 * time.Second
	lockStaleAfter        = 3 * lockHeartbeatInterval // ~45s: 3 missed beats before a waiter takes over
)

// AcquireLock blocks until no other instance's heartbeat is fresh, then
// starts a background goroutine that re-writes the lock message every
// lockHeartbeatInterval until ctx is cancelled. Call before bot.Open() so a
// blocked instance never opens a gateway connection while waiting.
func AcquireLock(ctx context.Context, s *discordgo.Session, channelID, messageID, instanceID string) {
	if channelID == "" {
		slog.Warn("no lock channel configured (DEV_CHANNEL_ID), skipping instance lock")
		return
	}

	for messageID != "" {
		holder, ts, ok := readLock(s, channelID, messageID)
		if !ok || holder == instanceID || time.Since(ts) > lockStaleAfter {
			break
		}
		slog.Info("another bot instance holds the lock, waiting", "holder", holder, "age", time.Since(ts).Round(time.Second))
		select {
		case <-ctx.Done():
			return
		case <-time.After(lockHeartbeatInterval):
		}
	}

	messageID = writeLock(s, channelID, messageID, instanceID)
	slog.Info("acquired instance lock", "instance", instanceID)

	go func() {
		t := time.NewTicker(lockHeartbeatInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				writeLock(s, channelID, messageID, instanceID)
			}
		}
	}()
}

func lockContent(instanceID string, ts time.Time) string {
	return fmt.Sprintf("🔒 lock held by `%s` — heartbeat %d", instanceID, ts.Unix())
}

// parseLockContent extracts the holder name and heartbeat time from a lock
// message body previously written by lockContent.
func parseLockContent(content string) (holder string, ts time.Time, ok bool) {
	parts := strings.Split(content, "`")
	if len(parts) < 2 {
		return "", time.Time{}, false
	}
	i := strings.LastIndex(content, "heartbeat ")
	if i < 0 {
		return "", time.Time{}, false
	}
	unix, err := strconv.ParseInt(strings.TrimSpace(content[i+len("heartbeat "):]), 10, 64)
	if err != nil {
		return "", time.Time{}, false
	}
	return parts[1], time.Unix(unix, 0), true
}

func readLock(s *discordgo.Session, channelID, messageID string) (holder string, ts time.Time, ok bool) {
	m, err := s.ChannelMessage(channelID, messageID)
	if err != nil {
		slog.Warn("reading lock message", "err", err)
		return "", time.Time{}, false
	}
	return parseLockContent(m.Content)
}

// writeLock edits messageID in place, or creates it on the first run when
// empty, returning the message ID to reuse on subsequent heartbeats.
func writeLock(s *discordgo.Session, channelID, messageID, instanceID string) string {
	content := lockContent(instanceID, time.Now())
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
