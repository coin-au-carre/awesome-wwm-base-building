package discord

import (
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"

	"ruby/internal/generator"
	"ruby/internal/guild"
)

// PickRandomGuild returns a random guild that has at least one screenshot.
func PickRandomGuild(guilds []guild.Guild) (guild.Guild, string, bool) {
	var candidates []guild.Guild
	for _, g := range guilds {
		if len(g.Screenshots) > 0 {
			candidates = append(candidates, g)
		}
	}
	if len(candidates) == 0 {
		return guild.Guild{}, "", false
	}
	pick := candidates[rand.IntN(len(candidates))]
	imgURL := pick.Screenshots[rand.IntN(len(pick.Screenshots))]
	return pick, imgURL, true
}

// FormatSpotlightMessage builds the Discord message for a guild spotlight.
func FormatSpotlightMessage(g guild.Guild) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## 🏰 Guild Spotlight: **%s**\n", g.Name)
	fmt.Fprintf(&sb, "-# 🎲 Randomly picked from the list\n")
	var meta []string
	if len(g.Builders) > 0 {
		meta = append(meta, "👷 "+strings.Join(g.Builders, ", "))
	}
	if len(g.Tags) > 0 {
		meta = append(meta, "🏷️ "+strings.Join(g.Tags, ", "))
	}
	meta = append(meta, fmt.Sprintf("⭐ %d", g.Score))
	fmt.Fprintf(&sb, "%s\n", strings.Join(meta, " · "))
	websiteURL := fmt.Sprintf("%s/guilds/%s", generator.WebsiteBase, generator.Slugify(g.Name))
	if g.DiscordThread != "" {
		fmt.Fprintf(&sb, "🔗 %s · [Website](%s)", g.DiscordThread, websiteURL)
	} else {
		fmt.Fprintf(&sb, "🔗 [Website](%s)", websiteURL)
	}
	return sb.String()
}

// DownloadImage fetches an image URL and returns its body and a derived filename.
func DownloadImage(url string) (io.ReadCloser, string, error) {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("HTTP %d fetching image", resp.StatusCode)
	}
	parts := strings.Split(strings.Split(url, "?")[0], "/")
	name := parts[len(parts)-1]
	if name == "" {
		name = "screenshot.png"
	}
	return resp.Body, name, nil
}
