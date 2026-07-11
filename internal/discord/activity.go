package discord

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// ActivityState tracks per-channel pagination progress and lifetime message
// counts per user, so reruns only fetch messages posted since the last run.
type ActivityState struct {
	LastMessageID map[string]string `json:"lastMessageID"`
	Counts        map[string]int    `json:"counts"`
}

// LoadActivityState reads state from path. Missing file is treated as empty.
func LoadActivityState(path string) (*ActivityState, error) {
	s := &ActivityState{LastMessageID: map[string]string{}, Counts: map[string]int{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading activity state: %w", err)
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing activity state: %w", err)
	}
	return s, nil
}

// Save writes the state to disk.
func (s *ActivityState) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ChannelDelta is one channel's scan result: message counts per non-bot
// author since the stored boundary, and the newest message ID seen.
type ChannelDelta struct {
	ChannelID string
	Counts    map[string]int
	Newest    string
}

// collectChannelDelta fetches messages newer than boundary and tallies them
// per author. It touches no shared state, so callers can run it concurrently
// across channels and merge results afterwards.
func collectChannelDelta(session *discordgo.Session, channelID, boundary string) (ChannelDelta, error) {
	slog.Info("scanning channel", "channel", channelID, "resuming_after", boundary)
	delta := ChannelDelta{ChannelID: channelID, Counts: map[string]int{}}
	var before string
	page := 0
	counted := 0
	for {
		msgs, err := session.ChannelMessages(channelID, 100, before, "", "")
		if err != nil {
			return delta, fmt.Errorf("fetching messages for channel %s: %w", channelID, err)
		}
		page++
		if len(msgs) == 0 {
			break
		}
		if delta.Newest == "" {
			delta.Newest = msgs[0].ID
		}
		reachedBoundary := false
		for _, msg := range msgs {
			if msg.ID == boundary {
				reachedBoundary = true
				break
			}
			if msg.Author != nil && !msg.Author.Bot {
				delta.Counts[msg.Author.ID]++
				counted++
			}
		}
		slog.Info("fetched page", "channel", channelID, "page", page, "messages_this_page", len(msgs), "counted_so_far", counted)
		if reachedBoundary || len(msgs) < 100 {
			break
		}
		before = msgs[len(msgs)-1].ID
	}
	slog.Info("channel scan done", "channel", channelID, "pages", page, "new_messages_counted", counted)
	return delta, nil
}

// CollectActivity scans all given channels concurrently (one goroutine per
// channel — Discord's REST rate limit is per-route, so this is safe) and
// merges the results into s.
func CollectActivity(session *discordgo.Session, channelIDs []string, s *ActivityState) error {
	results := make([]ChannelDelta, len(channelIDs))
	errs := make([]error, len(channelIDs))
	var wg sync.WaitGroup
	for i, channelID := range channelIDs {
		wg.Add(1)
		go func(i int, channelID string) {
			defer wg.Done()
			results[i], errs[i] = collectChannelDelta(session, channelID, s.LastMessageID[channelID])
		}(i, channelID)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	for _, delta := range results {
		for userID, n := range delta.Counts {
			s.Counts[userID] += n
		}
		if delta.Newest != "" {
			s.LastMessageID[delta.ChannelID] = delta.Newest
		}
	}
	return nil
}
