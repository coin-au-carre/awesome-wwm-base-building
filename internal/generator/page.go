package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ruby/internal/guild"
)

// Slugify converts a guild name to a URL-safe filename.
func Slugify(name string) string {
	re := regexp.MustCompile(`[^\p{L}\p{N}]+`)
	return strings.Trim(re.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

func writePage(g *guild.Guild, dir string) error {
	content := buildPage(g, dir)
	path := filepath.Join(dir, Slugify(g.Name)+".md")
	return os.WriteFile(path, []byte(content), 0644)
}

func buildPage(g *guild.Guild, dir string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", g.Name))

	sb.WriteString("<table>\n")
	if g.ID != "" {
		sb.WriteString(fmt.Sprintf("  <tr><td>🆔 <b>Guild ID</b></td><td>%s</td></tr>\n", g.ID))
	}
	if len(g.Builders) > 0 {
		sb.WriteString(fmt.Sprintf("  <tr><td>🔨 <b>Builders</b></td><td>%s</td></tr>\n", strings.Join(g.Builders, ", ")))
	}
	if len(g.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("  <tr><td>🏷️ <b>Tags</b></td><td>%s</td></tr>\n", strings.Join(g.Tags, ", ")))
	}
	sb.WriteString(fmt.Sprintf("  <tr><td>⭐ <b>Score</b></td><td>%d</td></tr>\n", g.Score))
	if g.DiscordThread != "" {
		sb.WriteString(fmt.Sprintf(
			"  <tr><td>💬 <b>Discord</b></td><td><a href=%q>Join server</a> · <a href=%q>View thread</a></td></tr>\n",
			DiscordInvite, g.DiscordThread,
		))
	}
	sb.WriteString("</table>\n\n")

	if g.Lore != "" {
		sb.WriteString("## 📜 Lore\n\n" + g.Lore + "\n\n")
	}
	if g.WhatToVisit != "" {
		sb.WriteString("## 🗺️ What to Visit\n\n" + g.WhatToVisit + "\n\n")
	}

	sb.WriteString("---\n\n## 📸 Screenshots\n\n")
	if len(g.Screenshots) > 0 {
		for _, url := range g.Screenshots {
			sb.WriteString(fmt.Sprintf("![screenshot](%s)\n\n", url))
		}
	} else {
		sb.WriteString("*No screenshots available yet.*\n\n")
		sb.WriteString(fmt.Sprintf(
			"📸 **Want to showcase this guild base?**\n\n"+
				"[Join our Discord](%s) and post your screenshots in %s — "+
				"they will appear here automatically!\n",
			DiscordInvite, ShowcaseChannel,
		))
	}

	if g.BuilderDiscordID == "" || g.BuilderDiscordID == OwnerDiscordID {
		sb.WriteString("\n---\n\n")
		sb.WriteString(buildDiscordTemplate(g))
	}

	return sb.String()
}

func buildDiscordTemplate(g *guild.Guild) string {
	id := "YOUR_GUILD_ID"
	if g.ID != "" {
		id = g.ID
	}
	builders := "Builder1, Builder2"
	if len(g.Builders) > 0 {
		builders = strings.Join(g.Builders, ", ")
	}

	missing := []string{}
	if g.Lore == "" {
		missing = append(missing, "lore")
	}
	if len(g.Screenshots) == 0 {
		missing = append(missing, "screenshots")
	}

	var sb strings.Builder
	sb.WriteString("## 🏰 Is this your guild?\n\n")
	sb.WriteString(fmt.Sprintf(
		"**%s** is missing %s — if you're one of the builders, [join our Discord](%s) and post in %s to:\n\n",
		g.Name, strings.Join(missing, " and "), DiscordInvite, ShowcaseChannel,
	))
	sb.WriteString("- Add your lore & points of interest\n")
	sb.WriteString("- Upload screenshots\n")
	sb.WriteString("- Collect votes ⭐\n\n")
	sb.WriteString("<details>\n<summary>📋 Copy this template</summary>\n\n")
	sb.WriteString("```\n")
	sb.WriteString(fmt.Sprintf("## :japanese_castle: %s [%s]\n", g.Name, id))
	sb.WriteString(fmt.Sprintf(":construction_worker: Builders: %s\n", builders))
	sb.WriteString("\n### :pencil: Lore\nREPLACE_WITH_YOUR_LORE\n")
	sb.WriteString("\n### :mage: What to visit\nDESCRIBE_POINT_OF_INTEREST\n")
	sb.WriteString("\n:ballot_box: Vote with reactions:\n")
	sb.WriteString(":star: Best overall | :thumbsup: Good base | :fire: Amazing creativity\n")
	sb.WriteString("```\n\n")
	sb.WriteString("</details>\n")
	return sb.String()
}

func buildGenericDiscordTemplate() string {
	var sb strings.Builder
	sb.WriteString("<details>\n<summary>📋 Copy this template</summary>\n\n")
	sb.WriteString("```\n")
	sb.WriteString("## :japanese_castle: YOUR_GUILD_NAME [YOUR_GUILD_ID]\n")
	sb.WriteString(":construction_worker: Builders: Builder1, Builder2\n")
	sb.WriteString("\n### :pencil: Lore\nREPLACE_WITH_YOUR_LORE\n")
	sb.WriteString("\n### :mage: What to visit\nDESCRIBE_POINT_OF_INTEREST\n")
	sb.WriteString("\n:ballot_box: Vote with reactions:\n")
	sb.WriteString(":star: Best overall | :thumbsup: Good base | :fire: Amazing creativity\n")
	sb.WriteString("```\n\n")
	sb.WriteString("</details>\n")
	return sb.String()
}
