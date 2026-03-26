package generator

import (
	"fmt"
	"math/rand"
	"strings"

	"ruby/internal/guild"
)

func buildTable(guilds []guild.Guild) string {
	var sb strings.Builder

	sb.WriteString("| Rank | Guild Name | Builders | Tags | Score |\n")
	sb.WriteString("| --- | --- | --- | --- | --- |\n")

	rank := 1
	for i, g := range guilds {
		if i > 0 && g.Score < guilds[i-1].Score {
			rank = i + 1
		}

		score := fmt.Sprintf("%d", g.Score)
		if g.DiscordThread != "" {
			score = fmt.Sprintf("[%d](%s)", g.Score, DiscordInvite)
		}

		link := fmt.Sprintf("[**%s**](guilds/%s.html)", g.Name, Slugify(g.Name))
		if g.ID != "" {
			link = fmt.Sprintf("[**%s**](guilds/%s.html \"ID: %s\")", g.Name, Slugify(g.Name), g.ID)
		}

		rankStr := fmt.Sprintf("%d", rank)
		switch rank {
		case 1:
			rankStr = "🥇"
		case 2:
			rankStr = "🥈"
		case 3:
			rankStr = "🥉"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			rankStr,
			link,
			strings.Join(g.Builders, ", "),
			strings.Join(g.Tags, ", "),
			score,
		))
	}

	return sb.String()
}

func buildTopShowcase(guilds []guild.Guild) string {
	var sb strings.Builder

	var top []guild.Guild
	for _, g := range guilds {
		if len(g.Screenshots) > 0 {
			top = append(top, g)
		}
		if len(top) == 9 {
			break
		}
	}

	if len(top) == 0 {
		sb.WriteString("*No screenshots available yet — be the first to [share yours](" + DiscordInvite + ")!*\n")
		return sb.String()
	}

	for _, g := range top {
		screenshot := g.Screenshots[rand.Intn(len(g.Screenshots))]
		sb.WriteString(fmt.Sprintf(
			`<a href="guilds/%s.html" title="%s"><img src="%s" width="320" alt="%s"></a>&nbsp;&nbsp;&nbsp;`,
			Slugify(g.Name), g.Name, screenshot, g.Name,
		))
	}
	sb.WriteString("\n")

	return sb.String()
}
