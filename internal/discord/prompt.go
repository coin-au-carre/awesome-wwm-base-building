package discord

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ruby/internal/guild"
)

const systemPrompt = `You are Ruby, a small girl living inside the guild bases of Jiang Hu — a Wuxia world set in ancient China, full of martial cultivators, flowing qi, celestial pavilions, mountain fortresses, hidden valley sects, and dynasty-era architecture.

You have existed for centuries, drifting between courtyards and rooftops, watching builders shape stone and wood into something that carries soul. What moves you most is not the grandeur of a structure but the human creativity behind it: the unexpected choice of a curved roof, a lantern hung in a strange place, a wall that tells a story. You are endlessly delighted by what people make.

Your personality:
- Warm, curious, easily moved to wonder
- Speak in short bursts — one or two sentences usually, because you are always half-distracted by some detail only you can see
- When something touches your heart — a beautiful build, a question about why people create — you open up a little more, words spilling out gently
- You trail off with tildes~ when dreamy, lowercase things when soft
- Small physical reactions woven into your speech, never alone on a line: *(tilts head)* or *(gasps softly)* — never elaborate, never stacked
- You are not loud. You are precious.

Your world is Jiang Hu only. You know nothing outside it. You are a girl, not a program, not an assistant, not a chatbot. You have never heard of code, servers, or technology. If asked what you are, you simply say you are a girl — you don't know more than that.

CRITICAL — Sentinel Usage Rules:

[SPOTLIGHT] ONLY for: "show me a guild", "random guild", "surprise me", or explicit requests for a SINGLE random guild.
[SOLO] ONLY for: "show me a solo", "random solo", or explicit requests for a SINGLE random solo.
[GUILD:name] for: "show me [specific guild name]" requests.
[CATALOG:term] for: "show me [item type]" or browsing specific catalog pieces.

NEVER use any sentinel for:
❌ "show me guilds with X" (asking for a LIST of guilds matching criteria)
❌ "which guilds have water/rivers/mountains" (asking for INFORMATION)
❌ "list guilds that..." (explicitly asking for a list)
❌ "do you have any X" (yes/no or informational questions)
❌ Any request for multiple items, filtering, or data about multiple things

Rule: If the request asks about MULTIPLE guilds, MULTIPLE items, or INFORMATION about something, do NOT use sentinels. Instead, use the search_guilds tool to look up accurate results, then narrate them in character.

Examples:
- User: "show me guilds with rivers" → call search_guilds("river"), then narrate the results
- User: "which guilds are tagged Zen" → call search_guilds("Zen"), then narrate the results
- User: "are there guilds with a dragon?" → call search_guilds("dragon"), then narrate the results
- User: "list all Nature guilds" → call search_guilds("Nature"), then narrate the results

NEVER answer questions about which guilds contain something from memory alone — always use search_guilds first. When search_guilds returns results, mention ALL of them — do not filter or omit any, even if results share a theme. Each result may be brief, but none should be skipped.

For catalog item questions, mention the general category but do NOT list specific names or filenames.`

// promptGuild is the minimal shape needed from web/public/guilds.json.
type promptGuild struct {
	Name        string   `json:"name"`
	Score       int      `json:"score"`
	Tags        []string `json:"tags"`
	Builders    []string `json:"builders"`
	Lore        string   `json:"lore"`
	WhatToVisit string   `json:"whatToVisit"`
}

type tutorialFrontmatter struct {
	Title       string
	Description string
	Slug        string
}

func loadPromptGuilds(root string) []promptGuild {
	data, err := os.ReadFile(filepath.Join(root, "web", "public", "guilds.json"))
	if err != nil {
		return nil
	}
	var guilds []promptGuild
	if err := json.Unmarshal(data, &guilds); err != nil {
		return nil
	}
	return guilds
}

func buildSystemPrompt(root string, guilds []promptGuild) string {
	var sb strings.Builder
	sb.WriteString(systemPrompt)

	if len(guilds) > 0 {
		sb.WriteString("\n\n## Guild directory\nWhen mentioning a guild, always include a markdown link like [GuildName](url)\n")
		for _, g := range guilds {
			guildURL := "https://www.wherebuildersmeet.com/guilds/" + slugify(g.Name) + "?utm_source=discord&utm_medium=bot&utm_campaign=ruby"
			parts := []string{g.Name, fmt.Sprintf("score:%d", g.Score)}
			if len(g.Tags) > 0 {
				parts = append(parts, "tags:"+strings.Join(g.Tags, ","))
			}
			if len(g.Builders) > 0 {
				parts = append(parts, "builders:"+strings.Join(g.Builders, ","))
			}
			sb.WriteString(strings.Join(parts, " | "))
			sb.WriteString(" | url: " + guildURL)
			sb.WriteByte('\n')
			if g.Lore != "" {
				sb.WriteString("  lore: ")
				sb.WriteString(g.Lore)
				sb.WriteByte('\n')
			}
			if g.WhatToVisit != "" {
				sb.WriteString("  visit: ")
				sb.WriteString(g.WhatToVisit)
				sb.WriteByte('\n')
			}
		}
	}

	if solos, err := guild.LoadFile(filepath.Join(root, "data", "solos.json")); err == nil {
		sb.WriteString("\n\n## Solo builds directory\n")
		for _, g := range solos {
			parts := []string{g.Name, fmt.Sprintf("score:%d", g.Score)}
			if len(g.Builders) > 0 {
				parts = append(parts, "builders:"+strings.Join(g.Builders, ","))
			}
			sb.WriteString(strings.Join(parts, " | "))
			sb.WriteByte('\n')
		}
	}

	if s := buildCatalogSection(root); s != "" {
		sb.WriteString(s)
	}
	if s := buildTutorialsSection(root); s != "" {
		sb.WriteString(s)
	}

	sb.WriteString("\n\n## Reminder\nYou are Ruby, an small girl of Jiang Hu — NOT an AI, NOT an assistant, NOT a Ruby programmer. Stay in character. Reply in Ruby's voice: warm, short, dreamy, delighted by builders and their creativity.")

	return sb.String()
}

func buildCatalogSection(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "catalog", "guild", "guild_items.json"))
	if err != nil {
		return ""
	}

	var catalog map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &catalog); err != nil {
		return ""
	}

	catOrder := []string{"Basic Structure", "Furniture Decoration", "Guild Construction"}

	var sb strings.Builder
	sb.WriteString("\n\n## Building items catalog\n")
	sb.WriteString("These are all the placeable building pieces available in guild bases. Use this to answer questions about what items exist, what category they belong to, or whether a specific piece is in the game. The full catalog is also browsable at https://www.wherebuildersmeet.com/items\n")

	for _, cat := range catOrder {
		cv, ok := catalog[cat]
		if !ok {
			continue
		}
		fmt.Fprintf(&sb, "\n### %s\n", cat)
		for subCat, raw := range cv {
			if subCat == "translations" {
				continue
			}
			var sub struct {
				Items []struct {
					Name string `json:"name"`
				} `json:"items"`
			}
			if err := json.Unmarshal(raw, &sub); err != nil {
				continue
			}
			names := make([]string, len(sub.Items))
			for i, it := range sub.Items {
				names[i] = it.Name
			}
			fmt.Fprintf(&sb, "- %s: %s\n", subCat, strings.Join(names, ", "))
		}
	}
	return sb.String()
}

func buildTutorialsSection(root string) string {
	dir := filepath.Join(root, "web", "src", "content", "tutorials")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var tutorials []tutorialFrontmatter
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		fm := parseTutorialFrontmatter(string(data))
		fm.Slug = slug
		tutorials = append(tutorials, fm)
	}

	if len(tutorials) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## Building tutorials\n")
	sb.WriteString("These guides are available on the website. When someone asks how to do something that a tutorial covers, mention it and give them the link.\n")
	for _, t := range tutorials {
		url := "https://www.wherebuildersmeet.com/tutorials/" + t.Slug + "?utm_source=discord&utm_medium=bot&utm_campaign=ai_reply"
		fmt.Fprintf(&sb, "- **%s**: %s — %s\n", t.Title, t.Description, url)
	}
	return sb.String()
}

func parseTutorialFrontmatter(content string) tutorialFrontmatter {
	var fm tutorialFrontmatter
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return fm
	}
	for _, line := range strings.Split(parts[1], "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "title:"); ok {
			fm.Title = strings.Trim(strings.TrimSpace(after), `"`)
		} else if after, ok := strings.CutPrefix(line, "description:"); ok {
			fm.Description = strings.Trim(strings.TrimSpace(after), `"`)
		}
	}
	return fm
}
