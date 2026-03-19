package main

import (
	"encoding/json"
	"fmt"
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

	if err := injectTable(buildTable(guilds)); err != nil {
		fmt.Fprintf(os.Stderr, "error injecting table: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("done: %d guild pages updated, README.md table injected\n", len(guilds))
}

func buildTable(guilds []Guild) string {
	headers := []string{"Guild ID", "Guild Name", "Builders", "Tags", "Discord Thread", "Score"}

	rows := make([][]string, len(guilds))
	for i, g := range guilds {
		thread := ""
		if g.DiscordThread != "" {
			thread = fmt.Sprintf("[Join](%s)", g.DiscordThread)
		}
		rows[i] = []string{
			g.ID,
			fmt.Sprintf("[%s](%s/%s.md)", g.Name, guildsDir, slugify(g.Name)),
			strings.Join(g.Builders, ", "),
			strings.Join(g.Tags, ", "),
			thread,
			fmt.Sprintf("%d", g.Score),
		}
	}

	// compute column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	pad := func(s string, w int) string {
		return s + strings.Repeat(" ", w-len(s))
	}
	formatRow := func(cells []string) string {
		parts := make([]string, len(cells))
		for i, c := range cells {
			parts[i] = pad(c, widths[i])
		}
		return "| " + strings.Join(parts, " | ") + " |"
	}

	var sb strings.Builder
	sb.WriteString(formatRow(headers) + "\n")
	seps := make([]string, len(headers))
	for i, w := range widths {
		seps[i] = strings.Repeat("-", w)
	}
	sb.WriteString(formatRow(seps) + "\n")
	for _, row := range rows {
		sb.WriteString(formatRow(row) + "\n")
	}

	return sb.String()
}

func injectTable(table string) error {
	src, err := os.ReadFile("README.md")
	if err != nil {
		return fmt.Errorf("reading README.md: %w", err)
	}
	content := string(src)

	start := strings.Index(content, startMarker)
	end := strings.Index(content, endMarker)
	if start == -1 || end == -1 {
		return fmt.Errorf("markers not found in README.md — add %q and %q around the table", startMarker, endMarker)
	}
	if start > end {
		return fmt.Errorf("start marker appears after end marker in README.md")
	}

	updated := content[:start] +
		startMarker + "\n" +
		table +
		content[end:]

	return os.WriteFile("README.md", []byte(updated), 0644)
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
