// generate.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Guild struct {
	ID            string   `json:"id,omitempty"`
	Name          string   `json:"name"`
	Builders      []string `json:"builders"`
	Tags          []string `json:"tags,omitempty"`
	DiscordThread string   `json:"discordThread"`
	Lore          string   `json:"lore,omitempty"`
	WhatToVisit   string   `json:"whatToVisit,omitempty"`
	Score         int      `json:"score"`
	Screenshots   []string `json:"screenshots,omitempty"`
}

const (
	guildsDir            = "guilds"
	discordInvite        = "https://discord.gg/Qygt9u26Bn"
	showcaseChannel      = "`#base-guild-showcase`"
	introStart           = "<!-- INTRO_START -->"
	introEnd             = "<!-- INTRO_END -->"
	startMarker          = "<!-- GENERATED_TABLE_START -->"
	endMarker            = "<!-- GENERATED_TABLE_END -->"
	showcaseStart        = "<!-- TOP_SHOWCASE_START -->"
	showcaseEnd          = "<!-- TOP_SHOWCASE_END -->"
	lastUpdatedStart     = "<!-- LAST_UPDATED_START -->"
	lastUpdatedEnd       = "<!-- LAST_UPDATED_END -->"
	discordTemplateStart = "<!-- DISCORD_TEMPLATE_START -->"
	discordTemplateEnd   = "<!-- DISCORD_TEMPLATE_END -->"
)

func main() {
	clean := flag.Bool("clean", false, "remove guild pages that no longer exist in guilds.json")
	flag.Parse()

	data, err := os.ReadFile("guilds.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading guilds.json: %v\n", err)
		os.Exit(1)
	}

	var guilds []Guild
	if err := json.Unmarshal(data, &guilds); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	sort.Slice(guilds, func(i, j int) bool {
		return guilds[i].Score > guilds[j].Score
	})

	if *clean {
		if err := cleanGuildPages(guilds); err != nil {
			fmt.Fprintf(os.Stderr, "error cleaning guild pages: %v\n", err)
			os.Exit(1)
		}
	}

	if err := os.MkdirAll(guildsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating guilds dir: %v\n", err)
		os.Exit(1)
	}

	for i := range guilds {
		if err := generateGuildPage(&guilds[i]); err != nil {
			fmt.Fprintf(os.Stderr, "error generating page for %s: %v\n", guilds[i].Name, err)
		}
	}

	intro := fmt.Sprintf("🏰 **%d guild bases** listed — [add yours on Discord!](%s)", len(guilds), discordInvite)
	if err := injectBetweenMarkers("README.md", introStart, introEnd, intro); err != nil {
		fmt.Fprintf(os.Stderr, "error injecting intro: %v\n", err)
		os.Exit(1)
	}

	if err := injectBetweenMarkers("README.md", startMarker, endMarker, buildTable(guilds)); err != nil {
		fmt.Fprintf(os.Stderr, "error injecting table: %v\n", err)
		os.Exit(1)
	}

	if err := injectBetweenMarkers("README.md", showcaseStart, showcaseEnd, buildTopShowcase(guilds)); err != nil {
		fmt.Fprintf(os.Stderr, "error injecting top showcase: %v\n", err)
		os.Exit(1)
	}

	if err := injectBetweenMarkers("README.md", discordTemplateStart, discordTemplateEnd, buildGenericDiscordTemplate()); err != nil {
		fmt.Fprintf(os.Stderr, "error injecting discord template: %v\n", err)
		os.Exit(1)
	}

	lastUpdated := fmt.Sprintf("🔄 **Last synchronized:**  %s", time.Now().UTC().Format("January 2, 2006 at 15:04 UTC"))
	if err := injectBetweenMarkers("README.md", lastUpdatedStart, lastUpdatedEnd, lastUpdated); err != nil {
		fmt.Fprintf(os.Stderr, "error injecting last updated: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("done: %d guild pages updated, README.md table and showcase injected\n", len(guilds))
}

// buildTopShowcase returns a markdown snippet with the first screenshot of the
// top 3 guilds (by score) that have at least one screenshot.
func buildTopShowcase(guilds []Guild) string {
	var sb strings.Builder

	var top []Guild
	for _, g := range guilds {
		if len(g.Screenshots) > 0 {
			top = append(top, g)
		}
		if len(top) == 9 {
			break
		}
	}

	if len(top) == 0 {
		sb.WriteString("*No screenshots available yet — be the first to [share yours](" + discordInvite + ")!*\n")
		return sb.String()
	}

	for _, g := range top {
		screenshot := g.Screenshots[rand.Intn(len(g.Screenshots))]
		sb.WriteString(fmt.Sprintf(
			`<a href="%s/%s.html" title="%s"><img src="%s" width="320" alt="%s"></a>&nbsp;&nbsp;&nbsp;`,
			guildsDir, slugify(g.Name), g.Name,
			screenshot,
			g.Name,
		))
	}
	sb.WriteString("\n")

	return sb.String()
}

func buildTable(guilds []Guild) string {
	var sb strings.Builder

	sb.WriteString("| Guild Name | Builders | Tags | Discord Score |\n")
	sb.WriteString("| --- | --- | --- | --- |\n")

	for _, g := range guilds {
		score := fmt.Sprintf("%d", g.Score)
		if g.DiscordThread != "" {
			score = fmt.Sprintf("[%d](%s)", g.Score, g.DiscordThread)
		}
		link := fmt.Sprintf("[**%s**](%s/%s.html)", g.Name, guildsDir, slugify(g.Name))
		if g.ID != "" {
			link = fmt.Sprintf("[**%s**](%s/%s.html \"ID: %s\")", g.Name, guildsDir, slugify(g.Name), g.ID)
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			link,
			strings.Join(g.Builders, ", "),
			strings.Join(g.Tags, ", "),
			score,
		))
	}

	return sb.String()
}

func injectBetweenMarkers(file, startMk, endMk, content string) error {
	src, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	s := string(src)

	start := strings.Index(s, startMk)
	end := strings.Index(s, endMk)
	if start == -1 || end == -1 {
		return fmt.Errorf("markers %q / %q not found in %s", startMk, endMk, file)
	}
	if start > end {
		return fmt.Errorf("start marker appears after end marker in %s", file)
	}

	updated := s[:start] + startMk + "\n\n" + content + "\n" + s[end:]
	return os.WriteFile(file, []byte(updated), 0644)
}

func generateGuildPage(g *Guild) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", g.Name))

	// metadata block
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
		sb.WriteString(fmt.Sprintf("  <tr><td>💬 <b>Discord</b></td><td><a href=\"%s\">View thread</a></td></tr>\n", g.DiscordThread))
	}
	sb.WriteString("</table>\n\n")

	if g.Lore != "" {
		sb.WriteString("## 📜 Lore\n\n")
		sb.WriteString(g.Lore + "\n\n")
	}

	if g.WhatToVisit != "" {
		sb.WriteString("## 🗺️ What to Visit\n\n")
		sb.WriteString(g.WhatToVisit + "\n\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString("## 📸 Screenshots\n\n")

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
			discordInvite, showcaseChannel,
		))
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString(buildDiscordTemplate(g))

	filename := filepath.Join(guildsDir, slugify(g.Name)+".md")
	return os.WriteFile(filename, []byte(sb.String()), 0644)
}

func buildDiscordTemplate(g *Guild) string {
	id := "YOUR_GUILD_ID"
	if g.ID != "" {
		id = g.ID
	}

	builders := "Builder1, Builder2"
	if len(g.Builders) > 0 {
		builders = strings.Join(g.Builders, ", ")
	}

	var sb strings.Builder
	sb.WriteString("## 📋 Post Your base guild on Discord\n\n")
	sb.WriteString(fmt.Sprintf(
		"Is this your guild? [Join our Discord](%s) and post in %s to add screenshots and get votes!\n\n",
		discordInvite, showcaseChannel,
	))
	sb.WriteString("Copy and paste this template into your Discord thread:\n\n")
	sb.WriteString("```\n")
	sb.WriteString(fmt.Sprintf("## :japanese_castle: %s [%s]\n", g.Name, id))
	sb.WriteString(fmt.Sprintf(":construction_worker: Builders: %s\n", builders))
	sb.WriteString("\n### :pencil: Lore\n")
	sb.WriteString("REPLACE_WITH_YOUR_LORE\n")
	sb.WriteString("\n### :mage: What to visit\n")
	sb.WriteString("DESCRIBE_POINT_OF_INTEREST\n")
	sb.WriteString("\n:ballot_box: Vote with reactions:\n")
	sb.WriteString(":star: Best overall | :thumbsup: Good base | :fire: Amazing creativity\n")
	sb.WriteString("```\n")

	return sb.String()
}

func slugify(name string) string {
	re := regexp.MustCompile(`[^\p{L}\p{N}]+`)
	return strings.Trim(re.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

func buildGenericDiscordTemplate() string {
	var sb strings.Builder

	sb.WriteString("```\n")
	sb.WriteString("## :japanese_castle: YOUR_GUILD_NAME [YOUR_GUILD_ID]\n")
	sb.WriteString(":construction_worker: Builders: Builder1, Builder2\n")
	sb.WriteString("\n### :pencil: Lore\n")
	sb.WriteString("REPLACE_WITH_YOUR_LORE\n")
	sb.WriteString("\n### :mage: What to visit\n")
	sb.WriteString("DESCRIBE_POINT_OF_INTEREST\n")
	sb.WriteString("\n:ballot_box: Vote with reactions:\n")
	sb.WriteString(":star: Best overall | :thumbsup: Good base | :fire: Amazing creativity\n")
	sb.WriteString("```\n")

	return sb.String()
}

func cleanGuildPages(guilds []Guild) error {
	known := make(map[string]bool, len(guilds))
	for _, g := range guilds {
		known[slugify(g.Name)+".md"] = true
	}

	entries, err := os.ReadDir(guildsDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", guildsDir, err)
	}

	var removed int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "index.md" {
			continue
		}
		if !known[e.Name()] {
			path := filepath.Join(guildsDir, e.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing %s: %w", path, err)
			}
			fmt.Printf("removed stale guild page: %s\n", path)
			removed++
		}
	}

	fmt.Printf("clean: %d stale guild page(s) removed\n", removed)
	return nil
}
