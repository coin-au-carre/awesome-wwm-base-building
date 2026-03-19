package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Guild struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Builders      []string `json:"builders"`
	Tags          []string `json:"tags"`
	DiscordThread string   `json:"discordThread"`
	Score         int      `json:"score"`
}

func pad(s string, width int) string {
	return s + strings.Repeat(" ", width-len(s))
}

func main() {
	data, err := os.ReadFile("guilds.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading guilds.json: %v\n", err)
		os.Exit(1)
	}

	var guilds []Guild
	if err := json.Unmarshal(data, &guilds); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	sort.Slice(guilds, func(i, j int) bool {
		return guilds[i].Score > guilds[j].Score
	})

	headers := []string{"Guild ID", "Guild Name", "Builders", "Tags", "Discord Thread", "Score"}

	rows := make([][]string, len(guilds))
	for i, g := range guilds {
		discordThread := ""
		if g.DiscordThread != "" {
			discordThread = fmt.Sprintf("[Join](%s)", g.DiscordThread)
		}
		rows[i] = []string{
			g.ID,
			g.Name,
			strings.Join(g.Builders, ", "),
			strings.Join(g.Tags, ", "),
			discordThread,
			fmt.Sprintf("%d", g.Score),
		}
	}

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

	printRow := func(cells []string) {
		parts := make([]string, len(cells))
		for i, cell := range cells {
			parts[i] = pad(cell, widths[i])
		}
		fmt.Println("| " + strings.Join(parts, " | ") + " |")
	}

	printRow(headers)

	separators := make([]string, len(headers))
	for i, w := range widths {
		separators[i] = strings.Repeat("-", w)
	}
	printRow(separators)

	for _, row := range rows {
		printRow(row)
	}
}
