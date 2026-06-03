// cmd/patches-sync/main.go — fetch Google Sheets patch notes CSV → data/patches.json
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"

	"github.com/bwmarrin/discordgo"
)

var reDiscordMsgLink = regexp.MustCompile(`discord\.com/channels/\d+/(\d+)/(\d+)`)

const csvURL = "https://docs.google.com/spreadsheets/d/e/2PACX-1vQYRYkRj4HdlI1m7Sl4pfpyINlTW2GvwTtuUZJl2XnN0gbtR_S3OLg--Zk4a0td1NV8mz3ulUO8aU-4/pub?gid=1954115521&single=true&output=csv"

type Tip struct {
	Coolness    string   `json:"coolness"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Guild       bool     `json:"guild"`
	Solo        bool     `json:"solo"`
	PC          bool     `json:"pc"`
	Mobile      bool     `json:"mobile"`
	PS5         bool     `json:"ps5"`
	Media       []string `json:"media"`
	Version     string   `json:"version"`
	Notes       string   `json:"notes"`
}

func main() {
	root := flag.String("root", cmdutil.RootDir(), "root directory")
	dryRun := flag.Bool("dry-run", false, "fetch and parse but skip writing JSON")
	updatesChannelID := flag.String("updates-channel-id", os.Getenv("UPDATES_CHANNEL_ID"), "Discord channel ID for the updates digest message")
	updatesMessageID := flag.String("updates-message-id", os.Getenv("UPDATES_MESSAGE_ID"), "Discord message ID to edit (empty = create new)")
	flag.Parse()

	cmdutil.LoadEnv(*root)
	// Re-read env after LoadEnv in case vars were in .env
	if *updatesChannelID == "" {
		*updatesChannelID = os.Getenv("UPDATES_CHANNEL_ID")
	}
	if *updatesMessageID == "" {
		*updatesMessageID = os.Getenv("UPDATES_MESSAGE_ID")
	}
	token := "Bot " + cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	tips, err := fetchTips()
	if err != nil {
		slog.Error("fetching tips CSV", "err", err)
		os.Exit(1)
	}
	slog.Info("fetched tips", "count", len(tips))

	tips = resolveDiscordLinks(token, tips)
	tips = refreshMedia(token, tips)

	if *dryRun {
		slog.Info("dry-run: skipping write")
		return
	}

	out, err := json.MarshalIndent(tips, "", "  ")
	if err != nil {
		slog.Error("marshalling tips", "err", err)
		os.Exit(1)
	}

	dest := filepath.Join(*root, "data", "patches.json")
	if err := os.WriteFile(dest, out, 0o644); err != nil {
		slog.Error("writing patches.json", "err", err)
		os.Exit(1)
	}
	slog.Info("wrote patches.json", "path", dest, "count", len(tips))

	if *updatesChannelID != "" {
		postUpdatesDigest(token, *updatesChannelID, *updatesMessageID, tips)
	}
}

func postUpdatesDigest(token, channelID, messageID string, tips []Tip) {
	s, err := discordgo.New(token)
	if err != nil {
		slog.Error("creating Discord session for updates digest", "err", err)
		return
	}

	content := buildUpdatesContent(tips, true)
	if len(content) > 2000 {
		content = buildUpdatesContent(tips, false)
	}

	if messageID == "" {
		msg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: content,
			Flags:   discordgo.MessageFlagsSuppressEmbeds,
		})
		if err != nil {
			slog.Error("posting updates digest", "err", err)
			return
		}
		slog.Info("posted updates digest — add this to UPDATES_MESSAGE_ID env var", "messageID", msg.ID)
		return
	}

	_, err = s.ChannelMessageEdit(channelID, messageID, content)
	if err != nil {
		slog.Error("editing updates digest", "err", err)
		return
	}
	slog.Info("updated updates digest", "messageID", messageID)
}

// buildUpdatesContent builds the full-width plain-text message grouped by version.
// withDetails controls whether per-tip description/notes subtext lines are included.
func buildUpdatesContent(tips []Tip, withDetails bool) string {
	const (
		spreadsheetURL = "https://docs.google.com/spreadsheets/d/1JuRIdk45EEIVU3kxcpqNuGKZ_0LIKUskiC5q6Tqwi_E/edit?usp=sharing"
		websiteURL     = "https://www.wherebuildersmeet.com/updates/"
		maxMsg         = 1950
	)

	// Collect versions in order of first appearance.
	seen := make(map[string]bool)
	var versions []string
	grouped := make(map[string][]Tip)
	for _, t := range tips {
		if !seen[t.Version] {
			seen[t.Version] = true
			versions = append(versions, t.Version)
		}
		grouped[t.Version] = append(grouped[t.Version], t)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# 📰 Construction WWM Updates\n")
	fmt.Fprintf(&sb, "More on %s\n", websiteURL)
	fmt.Fprintf(&sb, "[Data comes from Super Sheet](%s) — help us keep it up to date!\n", spreadsheetURL)

	for _, ver := range versions {
		if versionBefore(ver, 1, 7) {
			continue
		}
		verTips := grouped[ver]
		highTips := make([]Tip, 0, len(verTips))
		for _, t := range verTips {
			if t.Coolness == "high" {
				highTips = append(highTips, t)
			}
		}
		if len(highTips) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "## %s\n", ver)
		shown := 0
		for _, t := range highTips {
			entry := tipEntry(t, withDetails)
			if sb.Len()+len(entry) > maxMsg {
				break
			}
			sb.WriteString(entry)
			shown++
		}
	}

	fmt.Fprintf(&sb, "\n-# Updated <t:%d:R>", time.Now().Unix())
	return sb.String()
}

func tipEntry(t Tip, withDetails bool) string {
	coolnessEmoji := map[string]string{
		"high":   "⭐",
		"normal": "🔹",
		"low":    "🔸",
	}
	emoji := coolnessEmoji[t.Coolness]
	if emoji == "" {
		emoji = "•"
	}

	line := fmt.Sprintf("%s **%s**%s\n", emoji, t.Title, tipPlatformTags(t))

	if withDetails {
		var extra string
		switch {
		case t.Description != "" && t.Notes != "":
			extra = truncateStr(t.Description, 90) + " — " + truncateStr(t.Notes, 70)
		case t.Description != "":
			extra = truncateStr(t.Description, 140)
		case t.Notes != "":
			extra = truncateStr(t.Notes, 140)
		}
		if extra != "" {
			line += fmt.Sprintf("-# %s\n", extra)
		}
	}
	return line
}

func tipPlatformTags(t Tip) string {
	if t.PC && t.Mobile && t.PS5 {
		return ""
	}
	var parts []string
	if t.PC {
		parts = append(parts, "PC")
	}
	if t.Mobile {
		parts = append(parts, "Mobile")
	}
	if t.PS5 {
		parts = append(parts, "PS5")
	}
	if len(parts) == 0 {
		return ""
	}
	return " · " + strings.Join(parts, " · ")
}

// versionBefore reports whether ver (e.g. "v1.6", "1.6") is before major.minor.
func versionBefore(ver string, major, minor int) bool {
	s := strings.TrimPrefix(ver, "v")
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	return maj < major || (maj == major && min < minor)
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func fetchTips() ([]Tip, error) {
	resp, err := http.Get(csvURL)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	rows, err := csv.NewReader(resp.Body).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}
	if len(rows) < 2 {
		return nil, nil
	}

	header := rows[0]
	idx := func(name string) int {
		for i, h := range header {
			if strings.TrimRight(strings.ToLower(strings.TrimSpace(h)), "* ") == name {
				return i
			}
		}
		return -1
	}
	col := func(row []string, name string) string {
		i := idx(name)
		if i < 0 || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	boolCol := func(row []string, name string) bool {
		return strings.EqualFold(col(row, name), "true")
	}

	var tips []Tip
	for _, row := range rows[1:] {
		title := col(row, "title")
		if title == "" {
			continue
		}
		media := []string{}
		if raw := col(row, "media"); raw != "" {
			for _, s := range strings.Split(raw, ";") {
				if u := strings.TrimSpace(s); u != "" {
					media = append(media, u)
				}
			}
		}
		tips = append(tips, Tip{
			Coolness:    strings.ToLower(col(row, "coolness")),
			Title:       title,
			Description: col(row, "description"),
			Guild:       boolCol(row, "guild"),
			Solo:        boolCol(row, "solo"),
			PC:          boolCol(row, "pc"),
			Mobile:      boolCol(row, "mobile"),
			PS5:         boolCol(row, "ps5"),
			Media:       media,
			Version:     col(row, "version"),
			Notes:       col(row, "notes"),
		})
	}
	return tips, nil
}

// resolveDiscordLinks replaces discord.com/channels/… message links with the
// actual CDN attachment URL fetched from the Discord API.
func resolveDiscordLinks(token string, tips []Tip) []Tip {
	type discordAttachment struct {
		URL string `json:"url"`
	}
	type discordMessage struct {
		Attachments []discordAttachment `json:"attachments"`
	}

	fetchAttachment := func(channelID, messageID string) string {
		url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s", channelID, messageID)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return ""
		}
		req.Header.Set("Authorization", token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return ""
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return ""
		}
		var msg discordMessage
		if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil || len(msg.Attachments) == 0 {
			return ""
		}
		return msg.Attachments[0].URL
	}

	cache := make(map[string]string)
	for i := range tips {
		for j, u := range tips[i].Media {
			m := reDiscordMsgLink.FindStringSubmatch(u)
			if m == nil {
				continue
			}
			channelID, messageID := m[1], m[2]
			cacheKey := channelID + "/" + messageID
			resolved, seen := cache[cacheKey]
			if !seen {
				resolved = fetchAttachment(channelID, messageID)
				cache[cacheKey] = resolved
				if resolved != "" {
					slog.Info("resolved Discord message link", "message", messageID, "url", resolved)
				} else {
					slog.Warn("could not resolve Discord message link", "url", u)
				}
			}
			if resolved != "" {
				tips[i].Media[j] = resolved
			}
		}
	}
	return tips
}

func refreshMedia(token string, tips []Tip) []Tip {
	var allURLs []string
	for _, tip := range tips {
		allURLs = append(allURLs, tip.Media...)
	}
	if len(allURLs) == 0 {
		return tips
	}

	cdnForms := make(map[string]string) // cdn form → original URL
	var toRefresh []string
	for _, orig := range allURLs {
		if !strings.Contains(orig, "discordapp.") {
			continue
		}
		cdn, err := discord.ToCDNForm(orig)
		if err != nil {
			slog.Warn("normalizing CDN URL", "url", orig, "err", err)
			continue
		}
		if _, seen := cdnForms[cdn]; !seen {
			cdnForms[cdn] = orig
			toRefresh = append(toRefresh, cdn)
		}
	}
	if len(toRefresh) == 0 {
		return tips
	}

	refreshed, err := discord.BulkRefreshURLs(token, toRefresh)
	if err != nil {
		slog.Warn("refreshing Discord CDN URLs — continuing without refresh", "err", err)
		return tips
	}
	slog.Info("refreshed Discord CDN URLs", "count", len(refreshed))

	replace := make(map[string]string)
	for cdn, orig := range cdnForms {
		if newURL, ok := refreshed[cdn]; ok {
			replace[orig] = newURL
		}
	}

	for i := range tips {
		for j, u := range tips[i].Media {
			if newURL, ok := replace[u]; ok {
				tips[i].Media[j] = newURL
			}
		}
	}
	return tips
}
