// cmd/tutorial/main.go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var reURL = regexp.MustCompile(`discord\.com/channels/\d+/(\d+)`)
var reSlug = regexp.MustCompile(`[^\p{L}\p{N}]+`)
var reDiscordVideoMarker = regexp.MustCompile(`<!--\s*discord-video:(\d+)/(\d+)\s*-->`)
var reVideoSrc = regexp.MustCompile(`(<video\s[^>]*src=)"[^"]*"`)

func slugify(s string) string {
	return strings.Trim(reSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func main() {
	root := flag.String("root", cmdutil.RootDir(), "repository root directory")
	list := flag.String("list", "", "file with one Discord thread URL per line")
	refreshVideos := flag.Bool("refresh-videos", false, "refresh Discord CDN URLs in all articles")
	flag.Parse()

	var urls []string
	if *list != "" {
		f, err := os.Open(*list)
		if err != nil {
			slog.Error("opening list file", "err", err)
			os.Exit(1)
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			urls = append(urls, line)
		}
		f.Close()
	}
	urls = append(urls, flag.Args()...)

	if len(urls) == 0 && !*refreshVideos {
		fmt.Fprintln(os.Stderr, "usage: tutorial [-list <file>] [-refresh-videos] <discord-thread-url>...")
		os.Exit(1)
	}

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file")
	}

	bot, err := discord.NewBot(cmdutil.RequireEnv("RUBY_BOT_TOKEN"), "")
	if err != nil {
		slog.Error("creating bot", "err", err)
		os.Exit(1)
	}
	defer bot.Close()
	if err := bot.Open(); err != nil {
		slog.Error("opening session", "err", err)
		os.Exit(1)
	}

	for _, rawURL := range urls {
		if err := syncThread(bot.Session, *root, rawURL); err != nil {
			slog.Error("syncing thread", "url", rawURL, "err", err)
		}
	}

	if *refreshVideos {
		articlesDir := filepath.Join(*root, "web", "src", "content", "articles")
		if err := refreshVideoURLs(bot.Session, articlesDir); err != nil {
			slog.Error("refreshing video URLs", "err", err)
			os.Exit(1)
		}
	}
}

func syncThread(s *discordgo.Session, root, rawURL string) error {
	m := reURL.FindStringSubmatch(rawURL)
	if m == nil {
		return fmt.Errorf("invalid Discord thread URL: %s", rawURL)
	}
	threadID := m[1]

	thread, err := s.Channel(threadID)
	if err != nil {
		return fmt.Errorf("fetching thread: %w", err)
	}

	allMsgs := fetchAllMessages(s, threadID)
	if len(allMsgs) == 0 {
		return fmt.Errorf("no messages found in thread %s", threadID)
	}

	authorID := allMsgs[0].Author.ID
	authorName := allMsgs[0].Author.Username
	if mem, err := s.GuildMember(thread.GuildID, authorID); err == nil && mem.Nick != "" {
		authorName = mem.Nick
	}

	slug := slugify(thread.Name)
	var firstImageURL string
	var parts []string

	var groups [][]string
	for _, msg := range allMsgs {
		text := strings.TrimSpace(msg.Content)
		var group []string
		for _, att := range msg.Attachments {
			ext := mediaExt(att.URL)
			if isVideo(ext) {
				group = append(group, fmt.Sprintf(
					`<video src="%s" controls style="border-radius: 0.75rem; width: 100%%; max-width: 720px;"></video>`,
					att.URL,
				))
				slog.Info("linked video (Discord CDN)", "url", att.URL)
				continue
			}
			if firstImageURL == "" {
				firstImageURL = att.URL
			}
			group = append(group, fmt.Sprintf(
				`<img src="%s" alt="" style="border-radius: 0.75rem; width: 100%%; max-width: 480px;" />`,
				att.URL,
			))
			slog.Info("linked image (Discord CDN)", "url", att.URL)
		}
		if text != "" {
			group = append(group, text)
		}
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}
	var groupStrs []string
	for _, g := range groups {
		groupStrs = append(groupStrs, strings.Join(g, "\n\n"))
	}
	parts = append(parts, strings.Join(groupStrs, "\n\n---\n\n"))

	outPath := filepath.Join(root, "web", "src", "content", "articles", slug+".md")

	var existingFrontmatter string
	if data, err := os.ReadFile(outPath); err == nil {
		if fm := extractFrontmatter(string(data)); fm != "" {
			existingFrontmatter = fm
		}
	}

	var content string
	if existingFrontmatter != "" {
		fm := refreshFrontmatterImage(existingFrontmatter, firstImageURL)
		content = fmt.Sprintf("---\n%s---\n\n%s\n", fm, strings.Join(parts, "\n\n"))
	} else {
		content = buildMarkdown(thread.Name, authorName, firstImageURL, parts)
	}

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing article: %w", err)
	}

	fmt.Printf("wrote  %s\n", outPath)
	fmt.Printf("author %s\n", authorName)
	fmt.Println("note   images and videos embedded via Discord CDN URL — no local files")
	return nil
}

// fetchAllMessages pages through all thread messages and returns them in chronological order.
func fetchAllMessages(s *discordgo.Session, threadID string) []*discordgo.Message {
	var all []*discordgo.Message
	var lastID string
	for {
		msgs, err := s.ChannelMessages(threadID, 100, lastID, "", "")
		if err != nil || len(msgs) == 0 {
			break
		}
		all = append(all, msgs...)
		lastID = msgs[len(msgs)-1].ID
		if len(msgs) < 100 {
			break
		}
	}
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	return all
}

var reImageField = regexp.MustCompile(`(?m)^image:.*$`)

func refreshFrontmatterImage(fm, imageURL string) string {
	if imageURL == "" {
		return fm
	}
	replacement := fmt.Sprintf("image: %q", imageURL)
	if reImageField.MatchString(fm) {
		return reImageField.ReplaceAllString(fm, replacement)
	}
	return fm + replacement + "\n"
}

func extractFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return ""
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return ""
	}
	return strings.TrimSpace(parts[1]) + "\n"
}

func buildMarkdown(title, author, imageURL string, parts []string) string {
	date := time.Now().Format("2006-01-02")
	imageField := ""
	if imageURL != "" {
		imageField = fmt.Sprintf("\nimage: %q", imageURL)
	}
	return fmt.Sprintf("---\ntitle: %q\ndescription: \"\"\ntags: []\nauthors: [%q]\ndate: %s\norder: 99%s\n---\n\n%s\n",
		title, author, date, imageField, strings.Join(parts, "\n\n"))
}

func mediaExt(rawURL string) string {
	clean := strings.Split(rawURL, "?")[0]
	parts := strings.Split(clean, "/")
	name := parts[len(parts)-1]
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return strings.ToLower(name[idx:])
	}
	return ".jpg"
}

func isVideo(ext string) bool {
	switch ext {
	case ".mp4", ".webm", ".mov", ".avi":
		return true
	}
	return false
}

// refreshVideoURLs scans all articles for <!-- discord-video:CHANNEL/MESSAGE --> markers
// and updates the src attribute of the following <video> tag with a fresh Discord CDN URL.
func refreshVideoURLs(s *discordgo.Session, articlesDir string) error {
	entries, err := os.ReadDir(articlesDir)
	if err != nil {
		return fmt.Errorf("reading articles dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(articlesDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("reading article", "file", e.Name(), "err", err)
			continue
		}
		content := string(data)
		updated, changed := replaceDiscordVideoSrcs(s, content)
		if !changed {
			continue
		}
		if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
			slog.Warn("writing article", "file", e.Name(), "err", err)
			continue
		}
		slog.Info("refreshed video URLs", "file", e.Name())
	}
	return nil
}

func replaceDiscordVideoSrcs(s *discordgo.Session, content string) (string, bool) {
	lines := strings.Split(content, "\n")
	changed := false
	for i, line := range lines {
		sub := reDiscordVideoMarker.FindStringSubmatch(line)
		if sub == nil {
			continue
		}
		channelID, messageID := sub[1], sub[2]
		msg, err := s.ChannelMessage(channelID, messageID)
		if err != nil {
			slog.Warn("fetching discord message for video", "channel", channelID, "message", messageID, "err", err)
			continue
		}
		var cdnURL string
		for _, att := range msg.Attachments {
			if isVideo(mediaExt(att.URL)) {
				cdnURL = att.URL
				break
			}
		}
		if cdnURL == "" {
			slog.Warn("no video attachment found", "channel", channelID, "message", messageID)
			continue
		}
		// Update src in the next <video> line (search up to 3 lines ahead).
		for j := i + 1; j < len(lines) && j <= i+3; j++ {
			if strings.Contains(lines[j], "<video") {
				lines[j] = reVideoSrc.ReplaceAllString(lines[j], `${1}"`+cdnURL+`"`)
				changed = true
				break
			}
		}
	}
	return strings.Join(lines, "\n"), changed
}

