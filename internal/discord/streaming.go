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
	mu     sync.Mutex
	root   string
	active map[string]*Streamer // keyed by userID
	saveCh chan struct{}
}

func NewStreamingTracker(root string) *StreamingTracker {
	t := &StreamingTracker{
		root:   root,
		active: make(map[string]*Streamer),
		saveCh: make(chan struct{}, 1),
	}
	go t.saveLoop()
	return t
}

func (t *StreamingTracker) HandleVoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	wasStreaming := e.BeforeUpdate != nil && e.BeforeUpdate.SelfStream
	isStreaming := e.SelfStream

	slog.Info("voice state update", "user", e.UserID, "channel", e.ChannelID, "self_stream", isStreaming, "was_stream", wasStreaming)

	if isStreaming == wasStreaming {
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
		slog.Info("streaming started", "user", username, "channel", channelName)
	} else {
		if streamer, ok := t.active[e.UserID]; ok {
			slog.Info("streaming stopped", "user", streamer.Username, "channel", streamer.ChannelName)
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

func (t *StreamingTracker) saveAndPush() {
	t.mu.Lock()
	streamers := make([]Streamer, 0, len(t.active))
	for _, s := range t.active {
		streamers = append(streamers, *s)
	}
	t.mu.Unlock()

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
		{"git", "push"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = t.root
		out, err := cmd.CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "nothing to commit") {
				return
			}
			slog.Error("git op failed", "args", args, "err", err, "output", string(out))
			return
		}
	}
	slog.Info("streaming.json pushed", "streamers", len(streamers))
}
