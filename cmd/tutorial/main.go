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

func slugify(s string) string {
	return strings.Trim(reSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func main() {
	root := flag.String("root", cmdutil.RootDir(), "repository root directory")
	list := flag.String("list", "", "file with one Discord thread URL per line")
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

	if len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "usage: tutorial [-list <file>] <discord-thread-url>...")
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

	for _, msg := range allMsgs {
		text := strings.TrimSpace(msg.Content)
		if text != "" {
			parts = append(parts, text)
		}
		for _, att := range msg.Attachments {
			ext := mediaExt(att.URL)
			if isVideo(ext) {
				parts = append(parts, fmt.Sprintf(
					`<video src="%s" controls style="border-radius: 0.75rem; width: 100%%;"></video>`,
					att.URL,
				))
				slog.Info("linked video (Discord CDN)", "url", att.URL)
				continue
			}
			if firstImageURL == "" {
				firstImageURL = att.URL
			}
			parts = append(parts, fmt.Sprintf(
				`<img src="%s" alt="" style="border-radius: 0.75rem; width: 100%%;" />`,
				att.URL,
			))
			slog.Info("linked image (Discord CDN)", "url", att.URL)
		}
	}

	outPath := filepath.Join(root, "web", "src", "content", "articles", slug+".md")

	var existingFrontmatter string
	if data, err := os.ReadFile(outPath); err == nil {
		if fm := extractFrontmatter(string(data)); fm != "" {
			existingFrontmatter = fm
		}
	}

	var content string
	if existingFrontmatter != "" {
		content = fmt.Sprintf("---\n%s---\n\n%s\n", existingFrontmatter, strings.Join(parts, "\n\n"))
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

