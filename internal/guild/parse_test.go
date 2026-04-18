package guild

import (
	"reflect"
	"testing"
)

func TestParseFirstPost(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantID       string
		wantGuild    string
		wantBuilders []string
		wantLore     string
		wantVisit    string
		wantCover    int
	}{
		{
			name: "standard colon format",
			content: `🏯 Iron Keep [12345678]

👷 Builders: Alice, Bob

📝 Lore

A fortress built on the edge of the world, where winter never ends.

🧙 What to visit

The great hall and the frozen courtyard.

🗳️ Vote`,
			wantID:       "12345678",
			wantGuild:    "Iron Keep",
			wantBuilders: []string{"Alice", "Bob"},
			wantLore:     "A fortress built on the edge of the world, where winter never ends.",
			wantVisit:    "The great hall and the frozen courtyard.",
		},
		{
			name: "equals format",
			content: `atma: 🏯 YourGuildName = Funcorp

👷 Builders: atma

📝 Lore = Hogwarts at WWM

🧙 What to visit = Students study magic, including Potions, Transfiguration, Charms, Defense Against the Dark Arts, and Herbology.

🗳️ Vote with reactions:
⭐ Best overall | 👍 Good base | 🔥 Amazing creativity `,
			wantGuild:    "Funcorp",
			wantBuilders: []string{"atma"},
			wantLore:     "Hogwarts at WWM",
			wantVisit:    "Students study magic, including Potions, Transfiguration, Charms, Defense Against the Dark Arts, and Herbology.",
		},
		{
			name: "bracket ID with parentheses",
			content: `🏯 Dragon's Lair (87654321)

👷 Builders: Zara`,
			wantID:       "87654321",
			wantGuild:    "Dragon's Lair",
			wantBuilders: []string{"Zara"},
		},
		{
			name:         "skip placeholder lore",
			content:      "👷 Builders: Solo\n\nLore\n\nreplace_with_your_lore\n\nWhat to visit\n\ndescribe_point_of_interest",
			wantBuilders: []string{"Solo"},
			wantLore:     "",
			wantVisit:    "",
		},
		{
			name:         "short lore is kept",
			content:      "👷 Builders: X\n\nLore\n\nToo short.\n\nWhat to visit\n\nAlso short.",
			wantBuilders: []string{"X"},
			wantLore:     "Too short.",
			wantVisit:    "Also short.",
		},
		{
			name: "cover index",
			content: `🏯 Keep [11223344]

👷 Builders: Dev

Cover: 3`,
			wantID:       "11223344",
			wantGuild:    "Keep",
			wantBuilders: []string{"Dev"},
			wantCover:    3,
		},
		{
			name: "single builder no comma",
			content: `🏯 Solo Base [55667788]

👷 Builder: Lone Wolf`,
			wantID:       "55667788",
			wantGuild:    "Solo Base",
			wantBuilders: []string{"Lone Wolf"},
		},
		{
			name:         "builder with bold label markdown",
			content:      "👷 **Builders:** Lanyueliang",
			wantBuilders: []string{"Lanyueliang"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, guild, builders, lore, visit, cover := ParseFirstPost(tt.content)
			if id != tt.wantID {
				t.Errorf("id: got %q, want %q", id, tt.wantID)
			}
			if guild != tt.wantGuild {
				t.Errorf("guildName: got %q, want %q", guild, tt.wantGuild)
			}
			if !reflect.DeepEqual(builders, tt.wantBuilders) {
				t.Errorf("builders: got %v, want %v", builders, tt.wantBuilders)
			}
			if lore != tt.wantLore {
				t.Errorf("lore: got %q, want %q", lore, tt.wantLore)
			}
			if visit != tt.wantVisit {
				t.Errorf("whatToVisit: got %q, want %q", visit, tt.wantVisit)
			}
			if cover != tt.wantCover {
				t.Errorf("coverIdx: got %d, want %d", cover, tt.wantCover)
			}
		})
	}
}

func TestExtractNameAndID(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantID   string
	}{
		{"🏯 Iron Keep - Season 2", "Iron Keep", ""},
		{"WITCHERS [10248427", "WITCHERS", "10248427"},
		{"Dragon's Lair (87654321)", "Dragon's Lair", "87654321"},
		{"  My Guild  ", "My Guild", ""},
		// Bare 8-digit number without brackets: stripped from name but not captured as ID
		{"Guild 12345678", "Guild", ""},
	}

	for _, tt := range tests {
		name, id := ExtractNameAndID(tt.input)
		if name != tt.wantName {
			t.Errorf("ExtractNameAndID(%q): name = %q, want %q", tt.input, name, tt.wantName)
		}
		if id != tt.wantID {
			t.Errorf("ExtractNameAndID(%q): id = %q, want %q", tt.input, id, tt.wantID)
		}
	}
}
