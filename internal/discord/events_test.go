package discord

import "testing"

func TestParseLocation(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantID   string
	}{
		// Bracket / paren with numeric ID
		{"Iron Fortress [12345678]", "Iron Fortress", "12345678"},
		{"Iron Fortress (12345678)", "Iron Fortress", "12345678"},
		{"Eden (ID 10000281)", "Eden", "10000281"},

		// Unclosed bracket
		{"Iron Fortress [12345678", "Iron Fortress", "12345678"},

		// Space-separated numeric ID
		{"Iron Fortress 12345678", "Iron Fortress", "12345678"},

		// Non-numeric in brackets — treated as plain name, no ID
		{"Iron Fortress [iron-fortress]", "Iron Fortress [iron-fortress]", ""},

		// Plain name, no ID
		{"Iron Fortress", "Iron Fortress", ""},
		{"My Cool Guild", "My Cool Guild", ""},

		// Empty
		{"", "", ""},
	}

	for _, tc := range tests {
		name, id := parseLocation(tc.input)
		if name != tc.wantName || id != tc.wantID {
			t.Errorf("parseLocation(%q) = (%q, %q), want (%q, %q)",
				tc.input, name, id, tc.wantName, tc.wantID)
		}
	}
}
