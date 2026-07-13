package discord

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadStreamingStatePreservesStartedAt(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "data"), 0755); err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	content := `{"streamers":[{"userID":"u1","username":"Alice","channelID":"c1","channelName":"Voice","startedAt":"` +
		started.Format(time.RFC3339) + `"}],"updatedAt":"` + started.Format(time.RFC3339) + `"}`
	if err := os.WriteFile(filepath.Join(root, "data", "streaming.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	active := loadStreamingState(root)
	s, ok := active["u1"]
	if !ok {
		t.Fatalf("expected user u1 to be seeded, got %v", active)
	}
	if !s.StartedAt.Equal(started) {
		t.Errorf("StartedAt = %v, want %v", s.StartedAt, started)
	}
}

func TestLoadStreamingStateMissingFile(t *testing.T) {
	active := loadStreamingState(t.TempDir())
	if len(active) != 0 {
		t.Errorf("expected empty map for missing file, got %v", active)
	}
}
