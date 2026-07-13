package discord

import (
	"testing"
	"time"
)

func TestParseLockContentRoundTrip(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	content := lockContent("vps-1", ts)

	holder, gotTs, ok := parseLockContent(content)
	if !ok {
		t.Fatalf("parseLockContent(%q) failed to parse", content)
	}
	if holder != "vps-1" {
		t.Errorf("holder = %q, want %q", holder, "vps-1")
	}
	if !gotTs.Equal(ts) {
		t.Errorf("ts = %v, want %v", gotTs, ts)
	}
}

func TestParseLockContentInvalid(t *testing.T) {
	if _, _, ok := parseLockContent("not a lock message"); ok {
		t.Error("expected ok=false for garbage content")
	}
}
