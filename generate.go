// generate.go
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Guild struct {
	ID            string   `json:"id,omitempty"`
	Name          string   `json:"name"`
	Builders      []string `json:"builders"`
	Tags          []string `json:"tags,omitempty"`
	DiscordThread string   `json:"discordThread"`
	Score         int      `json:"score"`
	Screenshots   []string `json:"screenshots,omitempty"`
}

const (
	guildsDir       = "guilds"
	discordInvite   = "https://discord.gg/Qygt9u26Bn"
	showcaseChannel = "`#base-guild-showcase`"
	startMarker     = "<!-- GENERATED_TABLE_START -->"
	endMarker       = "<!-- GENERATED_TABLE_END -->"
	showcaseStart   = "<!-- TOP_SHOWCASE_START -->"
	showcaseEnd     = "<!-- TOP_SHOWCASE_END -->"
)

func main() {
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

	if err := os.MkdirAll(guildsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating guilds dir: %v\n", err)
		os.Exit(1)
	}

	for i := range guilds {
		if err := generateGuildPage(&guilds[i]); err != nil {
			fmt.Fprintf(os.Stderr, "error generating page for %s: %v\n", guilds[i].Name, err)
		}
	}

	if err := injectBetweenMarkers("README.md", startMarker, endMarker, buildTable(guilds)); err != nil {
		fmt.Fprintf(os.Stderr, "error injecting table: %v\n", err)
		os.Exit(1)
	}

	if err := injectBetweenMarkers("README.md", showcaseStart, showcaseEnd, buildTopShowcase(guilds)); err != nil {
		fmt.Fprintf(os.Stderr, "error injecting top showcase: %v\n", err)
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
		if len(top) == 10 {
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
			`<a href="%s/%s.md" title="%s"><img src="%s" width="260" alt="%s"></a>&nbsp;&nbsp;&nbsp;`,
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

	sb.WriteString("| Guild ID | Guild Name | Builders | Tags | Discord Score |\n")
	sb.WriteString("| --- | --- | --- | --- | --- |\n")

	for _, g := range guilds {
		score := fmt.Sprintf("%d", g.Score)
		if g.DiscordThread != "" {
			score = fmt.Sprintf("[%d](%s)", g.Score, g.DiscordThread)
		}
		sb.WriteString(fmt.Sprintf("| %s | [**%s**](%s/%s.md) | %s | %s | %s |\n",
			g.ID,
			g.Name, guildsDir, slugify(g.Name),
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

	if g.ID != "" {
		sb.WriteString(fmt.Sprintf("**ID:** %s  \n", g.ID))
	}
	if len(g.Builders) > 0 {
		sb.WriteString(fmt.Sprintf("**Builders:** %s  \n", strings.Join(g.Builders, ", ")))
	}
	if len(g.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("**Tags:** %s  \n", strings.Join(g.Tags, ", ")))
	}
	sb.WriteString(fmt.Sprintf("**Score:** %d  \n", g.Score))
	if g.DiscordThread != "" {
		sb.WriteString(fmt.Sprintf("**Discord Thread:** [View](%s)  \n", g.DiscordThread))
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("## Screenshots\n\n")

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

	filename := filepath.Join(guildsDir, slugify(g.Name)+".md")
	return os.WriteFile(filename, []byte(sb.String()), 0644)
}

func slugify(name string) string {
	re := regexp.MustCompile(`[^\p{L}\p{N}]+`)
	return strings.Trim(re.ReplaceAllString(strings.ToLower(name), "-"), "-")
}
