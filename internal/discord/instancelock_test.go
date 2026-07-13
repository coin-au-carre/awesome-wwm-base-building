package discord

import (
	"testing"
	"time"
)

func TestParseLockContentRoundTrip(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	content := lockContent("vps-1", 1, ts)

	holder, priority, gotTs, ok := parseLockContent(content)
	if !ok {
		t.Fatalf("parseLockContent(%q) failed to parse", content)
	}
	if holder != "vps-1" {
		t.Errorf("holder = %q, want %q", holder, "vps-1")
	}
	if priority != 1 {
		t.Errorf("priority = %d, want %d", priority, 1)
	}
	if !gotTs.Equal(ts) {
		t.Errorf("ts = %v, want %v", gotTs, ts)
	}
}

func TestParseLockContentInvalid(t *testing.T) {
	if _, _, _, ok := parseLockContent("not a lock message"); ok {
		t.Error("expected ok=false for garbage content")
	}
}
