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
		{
			name: "AMP standard ### headers",
			content: `## 🏯 AMP [10185138]

👷 Builders: Pears, Laksa, Cakatoi, Smùid, Nanami, AshaAzazill, Ligesxila

### 📝 Lore
Forged not by war, but by friendship, laughter, and countless battles fought side by side, our guild stands as more than a clan — we are family. Whether we go by Kittens or Roaches, every member carries the same bond: loyalty, chaos, and a home that endures through every victory and defeat. In this world, names may change and legends may fade, but the family we built will remain.

### 🧙 What to visit
(1) Our AMP sign!
(2) Come to our Guild Party
(3) Nanami's Architectural designs!
(4) Our floating zen garden!
(5) Our lake!
(6) Our flower shop!`,
			wantID:       "10185138",
			wantGuild:    "AMP",
			wantBuilders: []string{"Pears", "Laksa", "Cakatoi", "Smùid", "Nanami", "AshaAzazill", "Ligesxila"},
			wantLore:     "Forged not by war, but by friendship, laughter, and countless battles fought side by side, our guild stands as more than a clan — we are family. Whether we go by Kittens or Roaches, every member carries the same bond: loyalty, chaos, and a home that endures through every victory and defeat. In this world, names may change and legends may fade, but the family we built will remain.",
			wantVisit:    "(1) Our AMP sign!\n(2) Come to our Guild Party\n(3) Nanami's Architectural designs!\n(4) Our floating zen garden!\n(5) Our lake!\n(6) Our flower shop!",
		},
		{
			name: "inline lore and what-to-visit on same line with backtick separator",
			content: `🏯 YourGuildName HoNK

👷 Builders: DrugzBunny

 📝 Lore Mountain Village Dragon Temple


🧙 What to visit` + "`" + ` Take a nice tour trough chill lake area before that u can spar in the huge Arena inspired by Gladiator. Then u can move up towards the mountain village and enjoy the nature, small huts and alot of places to chill. Near the lake u can visit the library. And last but not least the chills of the of our HoNK Dragon temple 🙂`,
			wantBuilders: []string{"DrugzBunny"},
			wantLore:     "Mountain Village Dragon Temple",
			wantVisit:    "Take a nice tour trough chill lake area before that u can spar in the huge Arena inspired by Gladiator. Then u can move up towards the mountain village and enjoy the nature, small huts and alot of places to chill. Near the lake u can visit the library. And last but not least the chills of the of our HoNK Dragon temple 🙂",
		},
		{
			name: "solo build free-form lore (no lore header)",
			content: `Building is probably one of my favorite things to do in the game. I enjoy working on projects of all sizes, from small corners to entire environments... But more than anything, I love giving them a purpose and a story that feels connected to the game's lore. So today, I'd like to humbly share one of my favorite creations, along with the story behind it. ♥️

(I also prepared a shortened English version of the original story with a little help from ChatGPT, so it wouldn't turn into an endless wall of text. 😄)

> Deep within the Sparkling Abyss, long before their names became forgotten legends, Li Zuo and Liu Qingyi lived a simple and happy life. Everything changed when the physician Sun Buqi subjected Qingyi to forbidden experiments. She survived, but at the cost of a poisoned immortality that slowly eroded her soul while leaving her body untouched.
>
> Knowing that she would eventually lose the very qualities that made her human, Qingyi wished to accept her fate and leave the world before she became incapable of love or compassion. Unable to let her go, Li Zuo chose another path. Deep within the Abyss, he built a refuge for her: lanterns hanging like stars, pavilions filled with paintings and books, floating gardens, and research halls, all created in the hope of giving her one final spring and perhaps a cure.
>
> For years, they studied the curse together. Li Zuo traveled the world to bring her its wonders, while Qingyi fought to preserve what remained of her humanity. Yet some wounds cannot be healed.
>
> When Qingyi realized that her soul was truly beginning to fade, she chose to depart while she could still love the world. On the night of her passing, the lanterns of the Abyss burned until dawn, and it is said that a white deer appeared to guide her spirit beyond the mortal realm.
>
> After her death, Li Zuo abandoned both hope and research. He built his own tomb in the depths of the Abyss and entrusted its protection to the King of Nothingness. Little by little, he turned away from the world of the living, consumed by a grief so profound that it left no room for light.
>
> Yet even now, on certain nights, lanterns still drift around a stone flower standing amid the dark waters. The elders claim that a lone figure can sometimes be seen dancing among the floating lights, as though the memories of Liu Qingyi refuse to fade away.`,
			wantLore: "Building is probably one of my favorite things to do in the game. I enjoy working on projects of all sizes, from small corners to entire environments... But more than anything, I love giving them a purpose and a story that feels connected to the game's lore. So today, I'd like to humbly share one of my favorite creations, along with the story behind it. ♥️\n\n(I also prepared a shortened English version of the original story with a little help from ChatGPT, so it wouldn't turn into an endless wall of text. 😄)\n\n> Deep within the Sparkling Abyss, long before their names became forgotten legends, Li Zuo and Liu Qingyi lived a simple and happy life. Everything changed when the physician Sun Buqi subjected Qingyi to forbidden experiments. She survived, but at the cost of a poisoned immortality that slowly eroded her soul while leaving her body untouched.\n>\n> Knowing that she would eventually lose the very qualities that made her human, Qingyi wished to accept her fate and leave the world before she became incapable of love or compassion. Unable to let her go, Li Zuo chose another path. Deep within the Abyss, he built a refuge for her: lanterns hanging like stars, pavilions filled with paintings and books, floating gardens, and research halls, all created in the hope of giving her one final spring and perhaps a cure.\n>\n> For years, they studied the curse together. Li Zuo traveled the world to bring her its wonders, while Qingyi fought to preserve what remained of her humanity. Yet some wounds cannot be healed.\n>\n> When Qingyi realized that her soul was truly beginning to fade, she chose to depart while she could still love the world. On the night of her passing, the lanterns of the Abyss burned until dawn, and it is said that a white deer appeared to guide her spirit beyond the mortal realm.\n>\n> After her death, Li Zuo abandoned both hope and research. He built his own tomb in the depths of the Abyss and entrusted its protection to the King of Nothingness. Little by little, he turned away from the world of the living, consumed by a grief so profound that it left no room for light.\n>\n> Yet even now, on certain nights, lanterns still drift around a stone flower standing amid the dark waters. The elders claim that a lone figure can sometimes be seen dancing among the floating lights, as though the memories of Liu Qingyi refuse to fade away.",
		},
		{
			// ✍️ is U+270D + U+FE0F (variation selector); the FE0F was causing reLore to fail to match
			name: "emoji with variation selector (FE0F) before lore and what to visit",
			content: `🏯 SHAMELESS [10008244]
👷‍♀️ Builder: Keishalily

 ✍️ Lore:
**Garden of Eden Resort**

A beautiful paradise awaits you at SHAMELESS.

💕   What to Visit:

Visits are HIGHLY encouraged!
🎋 Domed Library
⚔️ Tree suites

🗳️ Vote with Reactions!

Cover: 43`,
			wantID:       "10008244",
			wantGuild:    "SHAMELESS",
			wantBuilders: []string{"Keishalily"},
			wantLore:     "**Garden of Eden Resort**\n\nA beautiful paradise awaits you at SHAMELESS.",
			wantVisit:    "Visits are HIGHLY encouraged!\n🎋 Domed Library\n⚔️ Tree suites",
			wantCover:    43,
		},
		{
			name: "places to visit alias",
			content: `🏯 SNEJNAYA (10269444)

👷 Builders: Ðìana

Lore:
A snowy castle of the White Queen.

Places to visit:
•    The Rose Garden — where beauty and steel stand side by side.
•    The Castle Library — home to old chronicles.`,
			wantID:       "10269444",
			wantGuild:    "SNEJNAYA",
			wantBuilders: []string{"Ðìana"},
			wantLore:     "A snowy castle of the White Queen.",
			wantVisit:    "•    The Rose Garden — where beauty and steel stand side by side.\n•    The Castle Library — home to old chronicles.",
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
		input      string
		wantName   string
		wantID     string
		wantTitle  string
	}{
		{"🏯 Iron Keep - Season 2", "Iron Keep", "", "Season 2"},
		{"WITCHERS [10248427", "WITCHERS", "10248427", ""},
		{"Dragon's Lair (87654321)", "Dragon's Lair", "87654321", ""},
		{"  My Guild  ", "My Guild", "", ""},
		// Bare 8-digit number without brackets: stripped from name but not captured as ID
		{"Guild 12345678", "Guild", "", ""},
		// Colon separator — preferred format
		{"DrunkenFist: Winter Wonderland", "DrunkenFist", "", "Winter Wonderland"},
		{"DrunkenFist:Winter Wonderland", "DrunkenFist", "", "Winter Wonderland"},
		{"DrunkenFist : Winter Wonderland", "DrunkenFist", "", "Winter Wonderland"},
		{"🏯 DrunkenFist: Winter Wonderland", "DrunkenFist", "", "Winter Wonderland"},
		// ID at end of build title (after " -" separator)
		{"JadeHearth - The Gates Swing Open [10006818]", "JadeHearth", "10006818", "The Gates Swing Open"},
	}

	for _, tt := range tests {
		name, id, title := ExtractNameAndID(tt.input)
		if name != tt.wantName {
			t.Errorf("ExtractNameAndID(%q): name = %q, want %q", tt.input, name, tt.wantName)
		}
		if id != tt.wantID {
			t.Errorf("ExtractNameAndID(%q): id = %q, want %q", tt.input, id, tt.wantID)
		}
		if title != tt.wantTitle {
			t.Errorf("ExtractNameAndID(%q): buildTitle = %q, want %q", tt.input, title, tt.wantTitle)
		}
	}
}
