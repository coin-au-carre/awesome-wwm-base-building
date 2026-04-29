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

const websiteBase = "https://www.wherebuildersmeet.com"

func websiteURL(path, campaign string) string {
	return fmt.Sprintf("%s%s?utm_source=discord&utm_medium=bot&utm_campaign=%s", websiteBase, path, campaign)
}

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
// The cover image is preferred when set; otherwise a random screenshot is used.
func PickFromGuilds(guilds []guild.Guild) (guild.Guild, string, bool) {
	if len(guilds) == 0 {
		return guild.Guild{}, "", false
	}
	pick := guilds[rand.IntN(len(guilds))]
	imgURL := pick.CoverImage
	if imgURL == "" {
		imgURL = pick.Screenshots[rand.IntN(len(pick.Screenshots))]
	}
	return pick, imgURL, true
}

func buildGuildMeta(g guild.Guild, includeScore bool) []string {
	var meta []string
	if len(g.Builders) > 0 {
		meta = append(meta, "🎨 "+strings.Join(g.Builders, ", "))
	}
	if len(g.Tags) > 0 {
		meta = append(meta, "🏷️ "+strings.Join(g.Tags, ", "))
	}
	if includeScore {
		meta = append(meta, fmt.Sprintf("⭐ %d", g.Score))
	}
	return meta
}

func guildWebsiteURL(g guild.Guild, campaign string) string {
	name := g.GuildName
	if name == "" {
		name = g.Name
	}
	return websiteURL("/guilds/"+slugify(name), campaign)
}

func soloWebsiteURL(g guild.Guild, campaign string) string {
	name := g.GuildName
	if name == "" {
		name = g.Name
	}
	return websiteURL("/solos/"+slugify(name), campaign)
}

// FormatSpotlightMessage builds the Discord message for a guild spotlight.
// Pass random=true to include the "randomly picked" subtitle.
func FormatSpotlightMessage(g guild.Guild, random bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "**%s**\n", g.Name)
	if random {
		fmt.Fprintf(&sb, "-# 🎲 randomly picked\n")
	}
	meta := buildGuildMeta(g, false)
	if len(meta) > 0 {
		fmt.Fprintf(&sb, "%s\n", strings.Join(meta, " · "))
	}
	if g.DiscordThread != "" {
		fmt.Fprintf(&sb, "🔗 %s · [WBM page](%s)", g.DiscordThread, guildWebsiteURL(g, "spotlight"))
	} else {
		fmt.Fprintf(&sb, "🔗 [WBM page](%s)", guildWebsiteURL(g, "spotlight"))
	}
	return sb.String()
}

// FormatSoloSpotlightMessage builds the Discord message for a solo build spotlight.
func FormatSoloSpotlightMessage(g guild.Guild, random bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "**%s**\n", g.Name)
	if random {
		fmt.Fprintf(&sb, "-# 🎲 randomly picked\n")
	}
	meta := buildGuildMeta(g, false)
	if len(meta) > 0 {
		fmt.Fprintf(&sb, "%s\n", strings.Join(meta, " · "))
	}
	if g.DiscordThread != "" {
		fmt.Fprintf(&sb, "🔗 %s · [WBM page](%s)", g.DiscordThread, soloWebsiteURL(g, "spotlight"))
	} else {
		fmt.Fprintf(&sb, "🔗 [WBM page](%s)", soloWebsiteURL(g, "spotlight"))
	}
	return sb.String()
}

// FormatNewGuildMessage builds the Discord message announcing a newly discovered guild or solo build.
// Pass isSolo=true to use solo-build wording.
func FormatNewGuildMessage(g guild.Guild, isSolo bool) string {
	var sb strings.Builder
	if isSolo {
		fmt.Fprintf(&sb, "🏡 New solo build! Say hello to **%s**!\n", g.Name)
	} else {
		fmt.Fprintf(&sb, "🏯 New guild base! Say hello to **%s**!\n", g.Name)
	}
	meta := buildGuildMeta(g, false)
	if len(meta) > 0 {
		fmt.Fprintf(&sb, "%s\n", strings.Join(meta, " · "))
	}
	url := guildWebsiteURL(g, "new_guild")
	if isSolo {
		url = soloWebsiteURL(g, "new_guild")
	}
	if g.DiscordThread != "" {
		fmt.Fprintf(&sb, "🔗 %s · [Website](%s)", g.DiscordThread, url)
	} else {
		fmt.Fprintf(&sb, "🔗 [Website](%s)", url)
	}
	return sb.String()
}

// FormatMoreScreenshotsMessage builds the Discord message announcing that an existing
// guild or solo build has gained new screenshots.
func FormatMoreScreenshotsMessage(g guild.Guild, isSolo bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "📸 **%s** just got new screenshots!\n", g.Name)
	meta := buildGuildMeta(g, false)
	if len(meta) > 0 {
		fmt.Fprintf(&sb, "%s\n", strings.Join(meta, " · "))
	}
	url := guildWebsiteURL(g, "new_screenshots")
	if isSolo {
		url = soloWebsiteURL(g, "new_screenshots")
	}
	if g.DiscordThread != "" {
		fmt.Fprintf(&sb, "🔗 %s · [Website](<%s>)", g.DiscordThread, url)
	} else {
		fmt.Fprintf(&sb, "🔗 [Website](<%s>)", url)
	}
	return sb.String()
}

// FormatMoreVideosMessage builds the Discord message announcing that an existing
// guild or solo build has gained new videos.
func FormatMoreVideosMessage(g guild.Guild, isSolo bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "🎬 **%s** just got new videos!\n", g.Name)
	meta := buildGuildMeta(g, false)
	if len(meta) > 0 {
		fmt.Fprintf(&sb, "%s\n", strings.Join(meta, " · "))
	}
	url := guildWebsiteURL(g, "new_videos")
	if isSolo {
		url = soloWebsiteURL(g, "new_videos")
	}
	if g.DiscordThread != "" {
		fmt.Fprintf(&sb, "🔗 %s · [Website](<%s>)", g.DiscordThread, url)
	} else {
		fmt.Fprintf(&sb, "🔗 [Website](<%s>)", url)
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
