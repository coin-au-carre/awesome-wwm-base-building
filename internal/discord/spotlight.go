package discord

import (
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strings"

	"ruby/internal/guild"
)

const websiteBase = "https://coin-au-carre.github.io/awesome-wwm-base-building"

var reSlugify = regexp.MustCompile(`[^\p{L}\p{N}]+`)

func slugify(name string) string {
	return strings.Trim(reSlugify.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// PickGuildByName finds a guild whose name contains the given substring (case-insensitive)
// and returns it with a random screenshot URL. Returns false if none found.
func PickGuildByName(guilds []guild.Guild, name string) (guild.Guild, string, bool) {
	lower := strings.ToLower(name)
	var candidates []guild.Guild
	for _, g := range guilds {
		if strings.Contains(strings.ToLower(g.Name), lower) && len(g.Screenshots) > 0 {
			candidates = append(candidates, g)
		}
	}
	return PickFromGuilds(candidates)
}

// PickRandomGuild returns a random guild that has at least one screenshot.
func PickRandomGuild(guilds []guild.Guild) (guild.Guild, string, bool) {
	var candidates []guild.Guild
	for _, g := range guilds {
		if len(g.Screenshots) > 0 {
			candidates = append(candidates, g)
		}
	}
	return PickFromGuilds(candidates)
}

// PickFromGuilds returns a random guild and screenshot URL from the given slice (all assumed to have screenshots).
func PickFromGuilds(guilds []guild.Guild) (guild.Guild, string, bool) {
	if len(guilds) == 0 {
		return guild.Guild{}, "", false
	}
	pick := guilds[rand.IntN(len(guilds))]
	imgURL := pick.Screenshots[rand.IntN(len(pick.Screenshots))]
	return pick, imgURL, true
}

func buildGuildMeta(g guild.Guild, includeScore bool) []string {
	var meta []string
	if len(g.Builders) > 0 {
		meta = append(meta, "👷 "+strings.Join(g.Builders, ", "))
	}
	if len(g.Tags) > 0 {
		meta = append(meta, "🏷️ "+strings.Join(g.Tags, ", "))
	}
	if includeScore {
		meta = append(meta, fmt.Sprintf("⭐ %d", g.Score))
	}
	return meta
}

func guildWebsiteURL(g guild.Guild) string {
	return fmt.Sprintf("%s/guilds/%s", websiteBase, slugify(g.Name))
}

// FormatSpotlightMessage builds the Discord message for a guild spotlight.
// Pass random=true to include the "randomly picked" subtitle.
func FormatSpotlightMessage(g guild.Guild, random bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## 🏰 Guild Spotlight: **%s**\n", g.Name)
	if random {
		fmt.Fprintf(&sb, "-# 🎲 Randomly picked from the list\n")
	}
	meta := buildGuildMeta(g, true)
	fmt.Fprintf(&sb, "%s\n", strings.Join(meta, " · "))
	if g.DiscordThread != "" {
		fmt.Fprintf(&sb, "🔗 %s · [Website](%s)", g.DiscordThread, guildWebsiteURL(g))
	} else {
		fmt.Fprintf(&sb, "🔗 [Website](%s)", guildWebsiteURL(g))
	}
	return sb.String()
}

// FormatNewGuildMessage builds the Discord message announcing a newly discovered guild or solo build.
// Pass isSolo=true to use solo-build wording.
func FormatNewGuildMessage(g guild.Guild, isSolo bool) string {
	var sb strings.Builder
	if isSolo {
		fmt.Fprintf(&sb, "## 🆕 New Solo Build Discovered: **%s**\n", g.Name)
	} else {
		fmt.Fprintf(&sb, "## 🆕 New Guild Discovered: **%s**\n", g.Name)
	}
	fmt.Fprintf(&sb, "-# Just joined the list!\n")
	meta := buildGuildMeta(g, false)
	if len(meta) > 0 {
		fmt.Fprintf(&sb, "%s\n", strings.Join(meta, " · "))
	}
	if g.DiscordThread != "" {
		fmt.Fprintf(&sb, "🔗 %s · [Website](%s)", g.DiscordThread, guildWebsiteURL(g))
	} else {
		fmt.Fprintf(&sb, "🔗 [Website](%s)", guildWebsiteURL(g))
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
