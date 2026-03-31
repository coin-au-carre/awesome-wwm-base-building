package generator

import (
	"fmt"
	"net/url"
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

func tableRow(icon, label, value string) string {
	return fmt.Sprintf("  <tr><td>%s <b>%s</b></td><td>%s</td></tr>\n", icon, label, value)
}

func buildPage(g *guild.Guild, dir string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s\n\n", g.Name)

	sb.WriteString("<table>\n")
	if g.ID != "" {
		sb.WriteString(tableRow("🆔", "Guild ID", g.ID))
	}
	if g.GuildName != "" {
		sb.WriteString(tableRow("🏰", "Guild Name", g.GuildName))
	}
	if len(g.Builders) > 0 {
		sb.WriteString(tableRow("🔨", "Builders", strings.Join(g.Builders, ", ")))
	}
	if len(g.Tags) > 0 {
		sb.WriteString(tableRow("🏷️", "Tags", strings.Join(g.Tags, ", ")))
	}
	sb.WriteString(tableRow("⭐", "Score", fmt.Sprintf("%d", g.Score)))
	if g.DiscordThread != "" {
		sb.WriteString(tableRow("💬", "Discord", fmt.Sprintf(
			"<a href=%q>Join server</a> · <a href=%q>View thread</a>",
			DiscordInvite, g.DiscordThread,
		)))
	}
	sb.WriteString("</table>\n\n")

	if g.Lore != "" {
		sb.WriteString("## 📜 Lore\n\n" + g.Lore + "\n\n")
	}
	if g.WhatToVisit != "" {
		sb.WriteString("## 🗺️ What to Visit\n\n" + strings.ReplaceAll(g.WhatToVisit, "\n", "  \n") + "\n\n")
	}

	hasVideos := len(g.Videos) > 0
	hasShots := len(g.Screenshots) > 0
	hasBoth := hasVideos && hasShots

	sb.WriteString("---\n\n## 🎬 Media\n\n")
	if hasVideos {
		if hasBoth {
			sb.WriteString("### 🎥 Videos\n\n")
		}
		for _, v := range g.Videos {
			sb.WriteString(renderVideo(v))
		}
	}
	if hasShots {
		if hasBoth {
			sb.WriteString("### 📸 Screenshots\n\n")
		}
		for _, url := range g.Screenshots {
			fmt.Fprintf(&sb, "![screenshot](%s)\n\n", url)
		}
	}
	if !hasVideos && !hasShots {
		sb.WriteString("*No media available yet.*\n\n")
		fmt.Fprintf(&sb,
			"📸 **Want to showcase this guild base?**\n\n"+
				"[Join our Discord](%s) and post your screenshots in %s — "+
				"they will appear here automatically!\n",
			DiscordInvite, ShowcaseChannel,
		)
	}

	if g.BuilderDiscordID == "" || g.BuilderDiscordID == OwnerDiscordID {
		sb.WriteString("\n---\n\n")
		sb.WriteString(buildDiscordTemplate(g))
	}

	return sb.String()
}

func discordTemplateBody(name, id, builders string) string {
	return fmt.Sprintf(
		"<details>\n<summary>📋 Copy this template</summary>\n\n"+
			"<pre>\n"+
			"## :japanese_castle: %s [%s]\n"+
			":construction_worker: Builders: %s\n"+
			"\n### :pencil: Lore\nLore of the guild to fill if any...\n"+
			"\n### :mage: What to visit\nPoints of interests to fill...\n"+
			"\n:ballot_box: Vote with reactions:\n"+
			":star: Best overall | :thumbsup: Good base | :fire: Amazing creativity\n"+
			"</pre>\n\n"+
			"</details>\n",
		name, id, builders,
	)
}

func buildDiscordTemplate(g *guild.Guild) string {
	id := "guild_id"
	if g.ID != "" {
		id = g.ID
	}
	templateName := g.Name
	if g.GuildName != "" {
		templateName = g.GuildName
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
	fmt.Fprintf(&sb,
		"**%s** is missing %s — if you're one of the builders, [join our Discord](%s) and post in %s to:\n\n",
		g.Name, strings.Join(missing, " and "), DiscordInvite, ShowcaseChannel,
	)
	sb.WriteString("- Add your lore & points of interest\n")
	sb.WriteString("- Upload screenshots\n")
	sb.WriteString("- Collect votes ⭐\n\n")
	sb.WriteString(discordTemplateBody(templateName, id, builders))
	return sb.String()
}

// renderVideo returns an HTML snippet for a video URL.
// YouTube links are embedded as iframes; direct video files use <video>.
func renderVideo(rawURL string) string {
	if id := youtubeVideoID(rawURL); id != "" {
		return fmt.Sprintf(
			"<iframe width=\"100%%\" src=\"https://www.youtube.com/embed/%s\" frameborder=\"0\" allowfullscreen></iframe>\n\n",
			id,
		)
	}
	return fmt.Sprintf("<video controls src=%q width=\"100%%\"></video>\n\n", rawURL)
}

// youtubeVideoID extracts the video ID from a YouTube URL, or returns "".
func youtubeVideoID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	switch u.Host {
	case "youtu.be":
		return strings.TrimPrefix(u.Path, "/")
	case "www.youtube.com", "youtube.com":
		return u.Query().Get("v")
	}
	return ""
}

func buildGenericDiscordTemplate() string {
	return discordTemplateBody("YOUR_GUILD_NAME", "YOUR_GUILD_ID", "Builder1, Builder2")
}
