package discord

import (
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Streamer holds info about a user currently streaming in a voice channel.
type Streamer struct {
	UserID      string    `json:"userID"`
	Username    string    `json:"username"`
	ChannelID   string    `json:"channelID"`
	ChannelName string    `json:"channelName"`
	StartedAt   time.Time `json:"startedAt"`
}

type streamingFile struct {
	Streamers []Streamer `json:"streamers"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// StreamingTracker tracks active voice-channel streamers and persists state to
// data/streaming.json, committing and pushing on each change.
type StreamingTracker struct {
	mu      sync.Mutex
	root    string
	session *discordgo.Session
	guildID string
	active  map[string]*Streamer // keyed by userID
	saveCh  chan struct{}
}

func NewStreamingTracker(root string, session *discordgo.Session, guildID string) *StreamingTracker {
	t := &StreamingTracker{
		root:    root,
		session: session,
		guildID: guildID,
		active:  loadStreamingState(root),
		saveCh:  make(chan struct{}, 1),
	}
	go t.saveLoop()
	return t
}

// loadStreamingState seeds the tracker from the last-persisted streaming.json
// so a restart (e.g. a lock handoff between local and VPS) preserves each
// streamer's original StartedAt instead of resetting it to time.Now().
func loadStreamingState(root string) map[string]*Streamer {
	active := make(map[string]*Streamer)
	data, err := os.ReadFile(filepath.Join(root, "data", "streaming.json"))
	if err != nil {
		return active
	}
	var state streamingFile
	if err := json.Unmarshal(data, &state); err != nil {
		return active
	}
	for i := range state.Streamers {
		active[state.Streamers[i].UserID] = &state.Streamers[i]
	}
	return active
}

// HandleGuildCreate reconciles active streamers against the GUILD_CREATE
// voice states, the authoritative snapshot of who's streaming right now:
// carries over StartedAt for anyone still streaming, drops anyone who
// stopped while this instance wasn't connected, and only sets StartedAt to
// now for streamers seen for the first time.
func (t *StreamingTracker) HandleGuildCreate(s *discordgo.Session, e *discordgo.GuildCreate) {
	if t.guildID != "" && e.ID != t.guildID {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	reconciled := make(map[string]*Streamer)
	for _, vs := range e.VoiceStates {
		if !vs.SelfStream {
			continue
		}
		channelName := vs.ChannelID
		if ch, err := s.State.Channel(vs.ChannelID); err == nil {
			channelName = ch.Name
		} else if ch, err := s.Channel(vs.ChannelID); err == nil {
			channelName = ch.Name
		}
		username := vs.UserID
		if m, err := s.GuildMember(e.ID, vs.UserID); err == nil {
			switch {
			case m.Nick != "":
				username = m.Nick
			case m.User.GlobalName != "":
				username = m.User.GlobalName
			default:
				username = m.User.Username
			}
		}
		startedAt := time.Now().UTC()
		if existing, ok := t.active[vs.UserID]; ok {
			startedAt = existing.StartedAt
		}
		reconciled[vs.UserID] = &Streamer{
			UserID:      vs.UserID,
			Username:    username,
			ChannelID:   vs.ChannelID,
			ChannelName: channelName,
			StartedAt:   startedAt,
		}
		slog.Info("streaming detected on startup", "user", username, "userID", vs.UserID, "channel", channelName, "channelID", vs.ChannelID, "startedAt", startedAt)
	}
	t.active = reconciled

	select {
	case t.saveCh <- struct{}{}:
	default:
	}
}

func (t *StreamingTracker) HandleVoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	wasStreaming := e.BeforeUpdate != nil && e.BeforeUpdate.SelfStream
	isStreaming := e.SelfStream

	if isStreaming == wasStreaming {
		return
	}

	t.mu.Lock()
	guildID := t.guildID
	t.mu.Unlock()

	if guildID != "" && e.GuildID != guildID {
		slog.Info("streaming event from foreign guild, ignoring", "guild", e.GuildID, "user", e.UserID)
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if isStreaming {
		channelName := e.ChannelID
		if ch, err := s.State.Channel(e.ChannelID); err == nil {
			channelName = ch.Name
		} else if ch, err := s.Channel(e.ChannelID); err == nil {
			channelName = ch.Name
		}

		username := e.UserID
		if m, err := s.GuildMember(e.GuildID, e.UserID); err == nil {
			switch {
			case m.Nick != "":
				username = m.Nick
			case m.User.GlobalName != "":
				username = m.User.GlobalName
			default:
				username = m.User.Username
			}
		}

		t.active[e.UserID] = &Streamer{
			UserID:      e.UserID,
			Username:    username,
			ChannelID:   e.ChannelID,
			ChannelName: channelName,
			StartedAt:   time.Now().UTC(),
		}
		slog.Info("streaming started", "user", username, "userID", e.UserID, "channel", channelName, "channelID", e.ChannelID, "guild", e.GuildID)
	} else {
		if streamer, ok := t.active[e.UserID]; ok {
			slog.Info("streaming stopped", "user", streamer.Username, "userID", streamer.UserID, "channel", streamer.ChannelName)
		}
		delete(t.active, e.UserID)
	}

	// Non-blocking send — if a save is already queued, the latest state will be picked up.
	select {
	case t.saveCh <- struct{}{}:
	default:
	}
}

func (t *StreamingTracker) saveLoop() {
	for range t.saveCh {
		time.Sleep(5 * time.Second)
		t.saveAndPush()
	}
}

// activeEventChannelIDs returns the set of voice channel IDs currently hosting an active scheduled event.
func (t *StreamingTracker) activeEventChannelIDs() map[string]bool {
	if t.session == nil || t.guildID == "" {
		return nil
	}
	events, err := t.session.GuildScheduledEvents(t.guildID, false)
	if err != nil {
		slog.Warn("fetching scheduled events for streaming filter", "err", err)
		return nil
	}
	channels := map[string]bool{}
	for _, e := range events {
		if e.Status == discordgo.GuildScheduledEventStatusActive && e.ChannelID != "" {
			channels[e.ChannelID] = true
		}
	}
	return channels
}

func (t *StreamingTracker) saveAndPush() {
	t.mu.Lock()
	all := make([]Streamer, 0, len(t.active))
	for _, s := range t.active {
		all = append(all, *s)
	}
	t.mu.Unlock()

	eventChannels := t.activeEventChannelIDs()
	streamers := make([]Streamer, 0, len(all))
	for _, s := range all {
		if eventChannels[s.ChannelID] {
			slog.Info("streaming excluded: channel has active event", "user", s.Username, "channel", s.ChannelName)
			continue
		}
		streamers = append(streamers, s)
	}

	state := streamingFile{
		Streamers: streamers,
		UpdatedAt: time.Now().UTC(),
	}

	path := filepath.Join(t.root, "data", "streaming.json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		slog.Error("marshaling streaming state", "err", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		slog.Error("writing streaming.json", "err", err)
		return
	}

	for _, args := range [][]string{
		{"git", "add", "data/streaming.json"},
		{"git", "commit", "-m", "chore: live streaming update"},
		{"git", "pull", "--rebase"},
		{"git", "push"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = t.root
		out, err := cmd.CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "nothing to commit") {
				return
			}
			// A conflicting pull --rebase must never be left half-applied —
			// otherwise the next "git add" silently stages the file with raw
			// conflict markers still in it as if it were resolved.
			if args[1] == "pull" {
				abort := exec.Command("git", "rebase", "--abort")
				abort.Dir = t.root
				_ = abort.Run()
			}
			slog.Error("git op failed", "args", args, "err", err, "output", string(out))
			return
		}
	}
	slog.Info("streaming.json pushed", "streamers", len(streamers))
}
