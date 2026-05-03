package guild

import (
	"reflect"
	"testing"
)

func TestParseFirstPost(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		wantID          string
		wantGuild       string
		wantBuilders    []string
		wantLore        string
		wantVisit       string
		wantCover       int
		wantOnBehalf    string
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
		{
			name: "on behalf with @mention",
			content: `🏯 墨雨樓 [10029273]
👷 Builders: AcElDiaMon

Posted on behalf of @AcElDiaMon who kindly asked us`,
			wantID:       "10029273",
			wantGuild:    "墨雨樓",
			wantBuilders: []string{"AcElDiaMon"},
			wantOnBehalf: "AcElDiaMon",
		},
		{
			name: "on behalf no bracket ID",
			content: `🏯 AfterFlame
Posting on behalf of @FoxiKate who kindly allowed us.

👷 Builders: FoxiKate`,
			wantBuilders: []string{"FoxiKate"},
			wantOnBehalf: "FoxiKate",
		},
		{
			name: "on behalf with Discord snowflake mention",
			content: `## 🏯 AfterFlame

Posting on behalf of <@1179100397466570884>  who kindly allowed us.

👷 Builders: FoxiKate`,
			wantBuilders: []string{"FoxiKate"},
			wantOnBehalf: "1179100397466570884",
		},
		{
			name: "on behalf snowflake with bracket ID and bold builders",
			content: `## 🏯 墨雨樓 [10029273]

👷 **Builders:** AcElDiaMon

Posted on behalf of <@423190009445613568> who kindly asked us

### 📝 Lore


### 🧙 What to visit

- Pixel art
- Floating restaurant`,
			wantID:       "10029273",
			wantGuild:    "墨雨樓",
			wantBuilders: []string{"AcElDiaMon"},
			wantVisit:    "- Pixel art\n- Floating restaurant",
			wantOnBehalf: "423190009445613568",
		},
		{
			name:         "on behalf present but no @username",
			content:      "👷 Builders: X\n\nPosted on behalf of the community.",
			wantBuilders: []string{"X"},
			wantOnBehalf: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ParseFirstPost(tt.content)
			if p.ID != tt.wantID {
				t.Errorf("id: got %q, want %q", p.ID, tt.wantID)
			}
			if p.GuildName != tt.wantGuild {
				t.Errorf("guildName: got %q, want %q", p.GuildName, tt.wantGuild)
			}
			if !reflect.DeepEqual(p.Builders, tt.wantBuilders) {
				t.Errorf("builders: got %v, want %v", p.Builders, tt.wantBuilders)
			}
			if p.Lore != tt.wantLore {
				t.Errorf("lore: got %q, want %q", p.Lore, tt.wantLore)
			}
			if p.WhatToVisit != tt.wantVisit {
				t.Errorf("whatToVisit: got %q, want %q", p.WhatToVisit, tt.wantVisit)
			}
			if p.CoverIdx != tt.wantCover {
				t.Errorf("coverIdx: got %d, want %d", p.CoverIdx, tt.wantCover)
			}
			if p.PostedOnBehalfOf != tt.wantOnBehalf {
				t.Errorf("postedOnBehalfOf: got %q, want %q", p.PostedOnBehalfOf, tt.wantOnBehalf)
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
		name, id, _ := ExtractNameAndID(tt.input)
		if name != tt.wantName {
			t.Errorf("ExtractNameAndID(%q): name = %q, want %q", tt.input, name, tt.wantName)
		}
		if id != tt.wantID {
			t.Errorf("ExtractNameAndID(%q): id = %q, want %q", tt.input, id, tt.wantID)
		}
	}
}
