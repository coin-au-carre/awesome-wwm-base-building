// cmd/bugs-sync/main.go — fetch Google Sheets bugs CSV → data/bugs.json
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
	"time"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"

	"github.com/bwmarrin/discordgo"
)

var reDiscordMsgLink = regexp.MustCompile(`discord\.com/channels/\d+/(\d+)/(\d+)`)

const csvURL = "https://docs.google.com/spreadsheets/d/e/2PACX-1vQYRYkRj4HdlI1m7Sl4pfpyINlTW2GvwTtuUZJl2XnN0gbtR_S3OLg--Zk4a0td1NV8mz3ulUO8aU-4/pub?gid=0&single=true&output=csv"

type Bug struct {
	Severity string   `json:"severity"`
	Title    string   `json:"title"`
	Details  string   `json:"details"`
	Guild    bool     `json:"guild"`
	Solo     bool     `json:"solo"`
	PC       bool     `json:"pc"`
	Mobile   bool     `json:"mobile"`
	PS5      bool     `json:"ps5"`
	Media    []string `json:"media"`
	Version  string   `json:"version"`
	Date     string   `json:"date"`
	Notes    string   `json:"notes"`
}

func main() {
	root := flag.String("root", cmdutil.RootDir(), "root directory")
	dryRun := flag.Bool("dry-run", false, "fetch and parse but skip writing JSON")
	bugsChannelID := flag.String("bugs-channel-id", os.Getenv("BUGS_CHANNEL_ID"), "Discord channel ID for the bugs digest message")
	bugsMessageID := flag.String("bugs-message-id", os.Getenv("BUGS_MESSAGE_ID"), "Discord message ID to edit (empty = create new)")
	flag.Parse()

	cmdutil.LoadEnv(*root)
	// Re-read env after LoadEnv in case vars were in .env
	if *bugsChannelID == "" {
		*bugsChannelID = os.Getenv("BUGS_CHANNEL_ID")
	}
	if *bugsMessageID == "" {
		*bugsMessageID = os.Getenv("BUGS_MESSAGE_ID")
	}
	token := "Bot " + cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	bugs, err := fetchBugs()
	if err != nil {
		slog.Error("fetching bugs CSV", "err", err)
		os.Exit(1)
	}
	slog.Info("fetched bugs", "count", len(bugs))

	bugs = resolveDiscordLinks(token, bugs)
	bugs = refreshMedia(token, bugs)

	if *dryRun {
		slog.Info("dry-run: skipping write")
		return
	}

	out, err := json.MarshalIndent(bugs, "", "  ")
	if err != nil {
		slog.Error("marshalling bugs", "err", err)
		os.Exit(1)
	}

	dest := filepath.Join(*root, "data", "bugs.json")
	if err := os.WriteFile(dest, out, 0o644); err != nil {
		slog.Error("writing bugs.json", "err", err)
		os.Exit(1)
	}
	slog.Info("wrote bugs.json", "path", dest, "count", len(bugs))

	if *bugsChannelID != "" {
		postBugsDigest(token, *bugsChannelID, *bugsMessageID, bugs)
	}
}

func postBugsDigest(token, channelID, messageID string, bugs []Bug) {
	s, err := discordgo.New(token)
	if err != nil {
		slog.Error("creating Discord session for bugs digest", "err", err)
		return
	}

	content := buildBugsContent(bugs, true)
	if len(content) > 2000 {
		content = buildBugsContent(bugs, false)
	}

	if messageID == "" {
		msg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: content,
			Flags:   discordgo.MessageFlagsSuppressEmbeds,
		})
		if err != nil {
			slog.Error("posting bugs digest", "err", err)
			return
		}
		slog.Info("posted bugs digest — add this to BUGS_MESSAGE_ID env var", "messageID", msg.ID)
		return
	}

	_, err = s.ChannelMessageEdit(channelID, messageID, content)
	if err != nil {
		slog.Error("editing bugs digest", "err", err)
		return
	}
	slog.Info("updated bugs digest", "messageID", messageID)
}

// buildBugsContent builds the full-width plain-text message.
// withDetails controls whether per-bug detail/notes subtext lines are included.
func buildBugsContent(bugs []Bug, withDetails bool) string {
	order := []string{"high", "normal", "low"}
	sectionLabel := map[string]string{
		"high":   "🔴 **High**",
		"normal": "🟡 **Normal**",
		"low":    "🔵 **Low**",
	}

	grouped := make(map[string][]Bug)
	for _, b := range bugs {
		grouped[b.Severity] = append(grouped[b.Severity], b)
	}

	bugWord := "bugs"
	if len(bugs) == 1 {
		bugWord = "bug"
	}

	const (
		spreadsheetURL = "https://docs.google.com/spreadsheets/d/1JuRIdk45EEIVU3kxcpqNuGKZ_0LIKUskiC5q6Tqwi_E/edit?usp=sharing"
		websiteURL     = "https://www.wherebuildersmeet.com/bugs/"
		maxMsg         = 1950
	)

	var sb strings.Builder
	fmt.Fprintf(&sb, "# 🐛 Bug Tracker — %d active %s\n", len(bugs), bugWord)
	fmt.Fprintf(&sb, "More details on %s. Know how to report bugs on WWM [here](%s)\n", websiteURL, websiteURL+"#how-to-report")
	fmt.Fprintf(&sb, "Discuss bugs and report in <#1483483683456286911>\n")
	fmt.Fprintf(&sb, "[Data comes from Super Sheet](%s): if you wanna help and contribute, let us know!\n", spreadsheetURL)

	for _, sev := range order {
		sevBugs := grouped[sev]
		if len(sevBugs) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "\n%s — %d\n", sectionLabel[sev], len(sevBugs))
		shown := 0
		for _, b := range sevBugs {
			entry := bugEntry(b, withDetails)
			if sb.Len()+len(entry) > maxMsg {
				break
			}
			sb.WriteString(entry)
			shown++
		}
		if shown < len(sevBugs) {
			fmt.Fprintf(&sb, "-# …and %d more\n", len(sevBugs)-shown)
		}
	}

	fmt.Fprintf(&sb, "\n-# Updated <t:%d:R>", time.Now().Unix())
	return sb.String()
}

func bugEntry(b Bug, withDetails bool) string {
	var versionStr string
	if v := b.Version; v != "" && v != "v1.7" {
		versionStr = fmt.Sprintf(" (%s)", v)
	}
	var platforms string
	if p := bugPlatformTags(b); p != "" {
		platforms = p
	}
	line := fmt.Sprintf("• **%s**%s%s\n", b.Title, versionStr, platforms)

	if withDetails {
		var extra string
		switch {
		case b.Details != "":
			extra = truncateStr(b.Details, 90)
		case b.Details != "":
			extra = truncateStr(b.Details, 140)
			// case b.Notes != "":
			// 	extra = truncateStr(b.Notes, 140)
		}
		if extra != "" {
			line += fmt.Sprintf("-# %s\n", extra)
		}
	}
	return line
}

func bugPlatformTags(b Bug) string {
	var parts []string
	if b.PC {
		parts = append(parts, "PC")
	}
	if b.Mobile {
		parts = append(parts, "Mobile")
	}
	if b.PS5 {
		parts = append(parts, "PS5")
	}
	if len(parts) == 0 {
		return ""
	}
	return " · " + strings.Join(parts, " · ")
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func fetchBugs() ([]Bug, error) {
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

	var bugs []Bug
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
		bugs = append(bugs, Bug{
			Severity: strings.ToLower(col(row, "severity")),
			Title:    title,
			Details:  col(row, "details"),
			Guild:    boolCol(row, "guild"),
			Solo:     boolCol(row, "solo"),
			PC:       boolCol(row, "pc"),
			Mobile:   boolCol(row, "mobile"),
			PS5:      boolCol(row, "ps5"),
			Media:    media,
			Version:  col(row, "version"),
			Date:     col(row, "date"),
			Notes:    col(row, "notes"),
		})
	}
	return bugs, nil
}

// resolveDiscordLinks replaces discord.com/channels/… message links with the
// actual CDN attachment URL fetched from the Discord API.
func resolveDiscordLinks(token string, bugs []Bug) []Bug {
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
	for i := range bugs {
		for j, u := range bugs[i].Media {
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
				bugs[i].Media[j] = resolved
			}
		}
	}
	return bugs
}

func refreshMedia(token string, bugs []Bug) []Bug {
	var allURLs []string
	for _, bug := range bugs {
		allURLs = append(allURLs, bug.Media...)
	}
	if len(allURLs) == 0 {
		return bugs
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
		return bugs
	}

	refreshed, err := discord.BulkRefreshURLs(token, toRefresh)
	if err != nil {
		slog.Warn("refreshing Discord CDN URLs — continuing without refresh", "err", err)
		return bugs
	}
	slog.Info("refreshed Discord CDN URLs", "count", len(refreshed))

	replace := make(map[string]string)
	for cdn, orig := range cdnForms {
		if newURL, ok := refreshed[cdn]; ok {
			replace[orig] = newURL
		}
	}

	for i := range bugs {
		for j, u := range bugs[i].Media {
			if newURL, ok := replace[u]; ok {
				bugs[i].Media[j] = newURL
			}
		}
	}
	return bugs
}
