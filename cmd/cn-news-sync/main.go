// cmd/cn-news-sync/main.go — scrape CN server news & announcements from yysls.cn → data/cn_news.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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
	dryRun := flag.Bool("dry-run", false, "fetch and parse but skip writing JSON")
	limit := flag.Int("limit", 20, "max items per category to fetch")
	channelID := flag.String("channel-id", "", "Discord channel ID to post digest (leave empty to skip)")
	flag.Parse()

	cmdutil.LoadEnv(*root)

	if *channelID == "" {
		*channelID = os.Getenv("CN_NEWS_CHANNEL_ID")
	}

	fetchedAt := time.Now().UTC().Format(time.RFC3339)

	news, err := scrape(newsURL, *limit)
	if err != nil {
		slog.Error("scraping news", "err", err)
		os.Exit(1)
	}
	slog.Info("scraped news", "count", len(news))

	announcements, err := scrape(announcementURL, *limit)
	if err != nil {
		slog.Error("scraping announcements", "err", err)
		os.Exit(1)
	}
	slog.Info("scraped announcements", "count", len(announcements))

	all := merge(news, announcements)
	for i := range all {
		all[i].FetchedAt = fetchedAt
	}

	if *dryRun {
		for _, item := range all {
			slog.Info("item", "category", item.Category, "date", item.Date, "title", item.Title, "url", item.URL)
		}
		slog.Info("dry-run: skipping write")
		return
	}

	out, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		slog.Error("marshalling cn_news", "err", err)
		os.Exit(1)
	}

	dest := filepath.Join(*root, "data", "cn_news.json")
	if err := os.WriteFile(dest, out, 0o644); err != nil {
		slog.Error("writing cn_news.json", "err", err)
		os.Exit(1)
	}
	slog.Info("wrote cn_news.json", "path", dest, "count", len(all))

	if *channelID != "" {
		token := "Bot " + cmdutil.RequireEnv("RUBY_BOT_TOKEN")
		postDigest(token, *channelID, all)
	}
}

func scrape(url string, limit int) ([]NewsItem, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
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
			raw := dm[1] // e.g. "20260612"
			if len(raw) == 8 {
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

func extract(re *regexp.Regexp, s string, group int) string {
	m := re.FindStringSubmatch(s)
	if m == nil || group >= len(m) {
		return ""
	}
	return m[group]
}

// merge combines news and announcements sorted by date descending.
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

func postDigest(token, channelID string, items []NewsItem) {
	s, err := discordgo.New(token)
	if err != nil {
		slog.Error("creating discord session", "err", err)
		return
	}

	content := buildDigest(items)
	_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: content,
		Flags:   discordgo.MessageFlagsSuppressEmbeds,
	})
	if err != nil {
		slog.Error("posting CN news digest", "err", err)
		return
	}
	slog.Info("posted CN news digest", "channel", channelID, "items", len(items))
}

func buildDigest(items []NewsItem) string {
	const maxMsg = 1950
	categoryEmoji := map[string]string{
		"新闻": "📰",
		"公告": "📢",
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# 🇨🇳 CN Server News — yysls.cn\n")
	fmt.Fprintf(&sb, "-# <t:%d:R>\n", time.Now().Unix())

	for _, item := range items {
		emoji := categoryEmoji[item.Category]
		if emoji == "" {
			emoji = "•"
		}
		line := fmt.Sprintf("%s **[%s](%s)**\n", emoji, item.Title, item.URL)
		if item.Summary != "" && item.Summary != item.Title {
			line += fmt.Sprintf("-# %s\n", truncate(item.Summary, 100))
		}
		if sb.Len()+len(line) > maxMsg {
			break
		}
		sb.WriteString(line)
	}

	return sb.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
