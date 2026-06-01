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
	"strings"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"
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
	flag.Parse()

	cmdutil.LoadEnv(*root)
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
