// cmd/cn-news-sync/main.go — scrape CN server news & announcements from yysls.cn → data/cn_news.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"ruby/internal/cmdutil"

	"github.com/bwmarrin/discordgo"
)

const (
	newsURL         = "https://www.yysls.cn/news/official/"
	announcementURL = "https://www.yysls.cn/news/update/"
)

var (
	reItem    = regexp.MustCompile(`(?s)<li>\s*<div class="news-container">(.*?)</li>`)
	reHref    = regexp.MustCompile(`href="([^"]+)"`)
	reLabel   = regexp.MustCompile(`<i class="news-label">(.*?)</i>`)
	reTitle   = regexp.MustCompile(`(?s)<p class="news-tit">.*?<i[^>]*>.*?</i>(.*?)</p>`)
	reSummary = regexp.MustCompile(`<p class="news-text">(.*?)</p>`)
	reImage   = regexp.MustCompile(`<img src="([^"]+)"`)
	reDateURL = regexp.MustCompile(`/(\d{8})/`)
)

type updatedItem struct {
	old NewsItem
	new NewsItem
}

type NewsItem struct {
	Category  string `json:"category"`
	Title     string `json:"title"`
	Summary   string `json:"summary,omitempty"`
	URL       string `json:"url"`
	Date      string `json:"date"`
	ImageURL  string `json:"imageUrl,omitempty"`
	FetchedAt string `json:"fetchedAt"`
}

func main() {
	root := flag.String("root", cmdutil.RootDir(), "root directory")
	dryRun := flag.Bool("dry-run", false, "fetch and parse but skip writing JSON or posting")
	limit := flag.Int("limit", 20, "max items per category to fetch")
	channelID := flag.String("channel-id", "", "Discord channel ID for new-item alerts (or CN_NEWS_CHANNEL_ID env)")
	flag.Parse()

	cmdutil.LoadEnv(*root)

	if *channelID == "" {
		*channelID = os.Getenv("CN_NEWS_CHANNEL_ID")
	}
	if *channelID == "" {
		*channelID = os.Getenv("RUBY_CHANNEL_ID")
	}

	dest := filepath.Join(*root, "data", "cn_news.json")
	existing := loadExisting(dest)
	seenByURL := make(map[string]int, len(existing)) // URL → index in existing
	for i, item := range existing {
		seenByURL[item.URL] = i
	}
	slog.Info("loaded existing", "count", len(existing))

	news, err := scrape(newsURL, *limit)
	if err != nil {
		slog.Error("scraping news", "err", err)
		os.Exit(1)
	}
	announcements, err := scrape(announcementURL, *limit)
	if err != nil {
		slog.Error("scraping announcements", "err", err)
		os.Exit(1)
	}

	all := merge(news, announcements)

	// Separate new items from updated ones (title or summary changed).
	fetchedAt := time.Now().UTC().Format(time.RFC3339)
	var fresh []NewsItem
	var updated []updatedItem
	for i := range all {
		idx, seen := seenByURL[all[i].URL]
		if !seen {
			all[i].FetchedAt = fetchedAt
			fresh = append(fresh, all[i])
			continue
		}
		old := existing[idx]
		if all[i].Title != old.Title || all[i].Summary != old.Summary {
			updated = append(updated, updatedItem{old: old, new: all[i]})
			existing[idx].Title = all[i].Title
			existing[idx].Summary = all[i].Summary
		}
	}
	slog.Info("new items", "count", len(fresh))
	slog.Info("updated items", "count", len(updated))

	if *dryRun {
		for _, item := range fresh {
			slog.Info("new", "category", item.Category, "date", item.Date, "title", item.Title)
		}
		for _, u := range updated {
			slog.Info("updated", "url", u.new.URL, "old_title", u.old.Title, "new_title", u.new.Title)
		}
		slog.Info("dry-run: skipping write and post")
		return
	}

	if len(fresh) == 0 && len(updated) == 0 {
		slog.Info("nothing new — skipping write and post")
		return
	}

	// Merge new items at the front, preserve existing (with any title/summary patches applied).
	merged := append(fresh, existing...)

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		slog.Error("marshalling cn_news", "err", err)
		os.Exit(1)
	}
	if err := os.WriteFile(dest, out, 0o644); err != nil {
		slog.Error("writing cn_news.json", "err", err)
		os.Exit(1)
	}
	slog.Info("wrote cn_news.json", "path", dest, "new", len(fresh), "updated", len(updated), "total", len(merged))

	if *channelID != "" {
		token := "Bot " + cmdutil.RequireEnv("RUBY_BOT_TOKEN")
		if len(fresh) > 0 {
			postNewItems(token, *channelID, fresh)
		}
		if len(updated) > 0 {
			postUpdatedItems(token, *channelID, updated)
		}
	}
}

func loadExisting(path string) []NewsItem {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var items []NewsItem
	if err := json.Unmarshal(data, &items); err != nil {
		slog.Warn("parsing existing cn_news.json", "err", err)
		return nil
	}
	return items
}

func scrape(pageURL string, limit int) ([]NewsItem, error) {
	resp, err := http.Get(pageURL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", pageURL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return parseItems(string(body), limit), nil
}

func parseItems(html string, limit int) []NewsItem {
	matches := reItem.FindAllStringSubmatch(html, -1)
	var items []NewsItem
	for _, m := range matches {
		block := m[1]
		href := extract(reHref, block, 1)
		label := extract(reLabel, block, 1)
		title := strings.TrimSpace(extract(reTitle, block, 1))
		summary := strings.TrimSpace(extract(reSummary, block, 1))
		imageURL := extract(reImage, block, 1)

		date := ""
		if dm := reDateURL.FindStringSubmatch(href); dm != nil {
			if raw := dm[1]; len(raw) == 8 {
				date = raw[:4] + "-" + raw[4:6] + "-" + raw[6:]
			}
		}
		if title == "" || href == "" {
			continue
		}
		items = append(items, NewsItem{
			Category: label,
			Title:    title,
			Summary:  summary,
			URL:      href,
			Date:     date,
			ImageURL: imageURL,
		})
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items
}

func merge(news, announcements []NewsItem) []NewsItem {
	all := append(news, announcements...)
	slices.SortStableFunc(all, func(a, b NewsItem) int {
		if a.Date > b.Date {
			return -1
		}
		if a.Date < b.Date {
			return 1
		}
		return 0
	})
	return all
}

func googleTranslateURL(articleURL string) string {
	return "https://translate.google.com/translate?sl=zh-CN&tl=en&u=" + url.QueryEscape(articleURL)
}

func postNewItems(token, channelID string, items []NewsItem) {
	s, err := discordgo.New(token)
	if err != nil {
		slog.Error("creating discord session", "err", err)
		return
	}

	categoryEmoji := map[string]string{
		"新闻": "📰",
		"公告": "📢",
	}
	categoryLabel := map[string]string{
		"新闻": "News",
		"公告": "Announcement",
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "## 🇨🇳 New CN server content spotted!\n")

	for _, item := range items {
		emoji := categoryEmoji[item.Category]
		if emoji == "" {
			emoji = "•"
		}
		label := categoryLabel[item.Category]
		if label == "" {
			label = item.Category
		}
		gtURL := googleTranslateURL(item.URL)

		line := fmt.Sprintf("%s **[%s](%s)**\n", emoji, item.Title, item.URL)
		line += fmt.Sprintf("-# %s · %s · [Read via Google Translate ↗](%s)\n", label, item.Date, gtURL)
		if item.Summary != "" && item.Summary != item.Title {
			line += fmt.Sprintf("-# _%s_\n", truncate(item.Summary, 120))
		}

		if sb.Len()+len(line) > 1950 {
			send(s, channelID, sb.String())
			sb.Reset()
		}
		sb.WriteString(line)
	}

	if sb.Len() > 0 {
		send(s, channelID, sb.String())
	}
	slog.Info("posted CN news alert", "channel", channelID, "items", len(items))
}

func postUpdatedItems(token, channelID string, items []updatedItem) {
	s, err := discordgo.New(token)
	if err != nil {
		slog.Error("creating discord session", "err", err)
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "## 🇨🇳 CN server article updated!\n")

	for _, u := range items {
		gtURL := googleTranslateURL(u.new.URL)
		line := fmt.Sprintf("✏️ **[%s](%s)**\n", u.new.Title, u.new.URL)
		line += fmt.Sprintf("-# was: _%s_ · [Read via Google Translate ↗](%s)\n", u.old.Title, gtURL)

		if sb.Len()+len(line) > 1950 {
			send(s, channelID, sb.String())
			sb.Reset()
		}
		sb.WriteString(line)
	}

	if sb.Len() > 0 {
		send(s, channelID, sb.String())
	}
	slog.Info("posted CN news update alert", "channel", channelID, "items", len(items))
}

func send(s *discordgo.Session, channelID, content string) {
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: content,
		Flags:   discordgo.MessageFlagsSuppressEmbeds,
	})
	if err != nil {
		slog.Error("posting message", "err", err)
	}
}

func extract(re *regexp.Regexp, s string, group int) string {
	m := re.FindStringSubmatch(s)
	if m == nil || group >= len(m) {
		return ""
	}
	return m[group]
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
