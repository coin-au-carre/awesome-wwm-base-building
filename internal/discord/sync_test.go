package discord

import "testing"

func TestSimilarGuildName(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		// Identical normalized names — already deduplicated upstream, must return false.
		{"CobraKai", "CobraKai", false},
		{"My Guild", "myguild", false},

		// Substring containment.
		{"The Witchers", "Witchers", true},
		{"witchers", "thewitchers", true},

		// Close enough (1 edit, minLen >= 5).
		{"Guilds", "Guild", true},   // 1 deletion, minLen=5, threshold=1
		{"Short", "Shirt", true},    // 1 substitution, minLen=5, threshold=1
		{"Guild", "Gulid", false},   // transposition = 2 edits in standard Levenshtein, minLen=5, threshold=1 → no match

		// Too far apart for their length.
		{"CobraKai", "COBAKA", false}, // distance=2, minLen=6, threshold=1 — false positive that prompted this test

		// Longer names tolerate 2 edits (minLen >= 10).
		{"BlackDragon", "BlackDragoon", true}, // distance=2, minLen=10, threshold=2

		// Empty / very short names.
		{"", "guild", false},
		{"abc", "xyz", false},
	}

	for _, tt := range tests {
		got := similarGuildName(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("similarGuildName(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
