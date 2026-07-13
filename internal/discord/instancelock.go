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

// RunLocked keeps this process alive for as long as ctx runs, cycling
// between waiting for the lock and holding it. Whenever it holds the lock it
// calls onAcquire(activeCtx) once; activeCtx is cancelled the moment the
// lock is lost (or ctx itself is cancelled) so onAcquire can stop the
// gateway connection and any background watchers, then onRelease runs to
// finish tearing them down. If a higher-priority instance takes the lock, or
// this one's own heartbeat goes stale, RunLocked releases and goes back to
// waiting instead of exiting the process — so a 24/7 VPS instance idles
// while a higher-priority local instance is up, and reclaims automatically
// once it goes away.
func RunLocked(ctx context.Context, s *discordgo.Session, channelID, messageID, instanceID string, priority int, onAcquire func(context.Context), onRelease func()) {
	if channelID == "" {
		slog.Warn("no lock channel configured (DEV_CHANNEL_ID), skipping instance lock")
		activeCtx, activeCancel := context.WithCancel(ctx)
		onAcquire(activeCtx)
		<-ctx.Done()
		activeCancel()
		onRelease()
		return
	}

	for ctx.Err() == nil {
		// Staleness is judged purely against this reader's own clock: has the
		// embedded heartbeat value changed recently, as observed here? Never
		// compare the holder's embedded timestamp directly against our wall
		// clock — local and VPS clocks aren't guaranteed to be in sync (WSL2
		// clocks in particular drift after sleep/resume), which would make a
		// perfectly fresh heartbeat look stale (or vice versa) and cause an
		// endless preempt/reclaim flap between instances.
		lastSeenUnix := int64(-1)
		lastChangeAt := time.Now()
		for messageID != "" {
			holder, holderPriority, ts, ok := readLock(s, channelID, messageID)
			if !ok || holder == instanceID || priority > holderPriority {
				break
			}
			if unix := ts.Unix(); unix != lastSeenUnix {
				lastSeenUnix = unix
				lastChangeAt = time.Now()
			}
			if time.Since(lastChangeAt) > lockStaleAfter {
				break
			}
			slog.Info("another bot instance holds the lock, waiting", "holder", holder, "priority", holderPriority, "unchanged_for", time.Since(lastChangeAt).Round(time.Second))
			select {
			case <-ctx.Done():
				return
			case <-time.After(lockHeartbeatInterval):
			}
		}
		if ctx.Err() != nil {
			return
		}

		messageID = writeLock(s, channelID, messageID, instanceID, priority)
		slog.Info("acquired instance lock", "instance", instanceID, "priority", priority)

		activeCtx, activeCancel := context.WithCancel(ctx)
		onAcquire(activeCtx)

		t := time.NewTicker(lockHeartbeatInterval)
	heartbeat:
		for {
			select {
			case <-ctx.Done():
				t.Stop()
				activeCancel()
				onRelease()
				return
			case <-t.C:
				if holder, _, _, ok := readLock(s, channelID, messageID); ok && holder != instanceID {
					slog.Info("lock taken over by another instance, releasing and waiting to reclaim", "new_holder", holder)
					break heartbeat
				}
				writeLock(s, channelID, messageID, instanceID, priority)
			}
		}
		t.Stop()
		activeCancel()
		onRelease()
	}
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
