// cmd/tutorial/main.go
package main

import (
	"flag"
	"fmt"
	"io"
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
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: tutorial <discord-thread-url>")
		os.Exit(1)
	}

	m := reURL.FindStringSubmatch(flag.Arg(0))
	if m == nil {
		fmt.Fprintln(os.Stderr, "invalid Discord thread URL")
		os.Exit(1)
	}
	threadID := m[1]

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

	thread, err := bot.Session.Channel(threadID)
	if err != nil {
		slog.Error("fetching thread", "err", err)
		os.Exit(1)
	}

	allMsgs := fetchAllMessages(bot.Session, threadID)
	if len(allMsgs) == 0 {
		slog.Error("no messages found in thread")
		os.Exit(1)
	}

	// Resolve author display name (server nickname preferred)
	authorID := allMsgs[0].Author.ID
	authorName := allMsgs[0].Author.Username
	if mem, err := bot.Session.GuildMember(thread.GuildID, authorID); err == nil && mem.Nick != "" {
		authorName = mem.Nick
	}

	tutorialsDir := filepath.Join(*root, "web", "public", "tutorials")
	slug := slugify(thread.Name)
	mediaCounter := 0
	var parts []string

	for _, msg := range allMsgs {
		text := strings.TrimSpace(msg.Content)
		if text != "" {
			parts = append(parts, text)
		}
		for _, att := range msg.Attachments {
			mediaCounter++
			ext := mediaExt(att.URL)
			if isVideo(ext) {
				parts = append(parts, fmt.Sprintf(
					`<video src="%s" controls style="border-radius: 0.75rem; width: 100%%;"></video>`,
					att.URL,
				))
				slog.Info("linked video (Discord CDN)", "url", att.URL)
				continue
			}
			filename := fmt.Sprintf("%s_%d%s", slug, mediaCounter, ext)
			if err := saveFile(att.URL, filepath.Join(tutorialsDir, filename)); err != nil {
				slog.Warn("downloading image", "url", att.URL, "err", err)
				continue
			}
			parts = append(parts, fmt.Sprintf(
				`<img src="/tutorials/%s" alt="" style="border-radius: 0.75rem; width: 100%%;" />`,
				filename,
			))
			slog.Info("saved image", "file", filename)
		}
	}

	outPath := filepath.Join(*root, "web", "src", "content", "articles", slug+".md")

	// Try to preserve existing frontmatter if file exists
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
		content = buildMarkdown(thread.Name, authorName, parts)
	}

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		slog.Error("writing article", "err", err)
		os.Exit(1)
	}

	fmt.Printf("wrote  %s\n", outPath)
	fmt.Printf("author %s\n", authorName)
	fmt.Printf("images %d (saved locally)\n", mediaCounter)
	fmt.Println("note   videos embedded via Discord CDN URL — no local files")
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

func buildMarkdown(title, author string, parts []string) string {
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("---\ntitle: %q\ndescription: \"\"\ntags: []\nauthors: [%q]\ndate: %s\norder: 99\n---\n\n%s\n",
		title, author, date, strings.Join(parts, "\n\n"))
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

func saveFile(url, dst string) error {
	body, _, err := discord.DownloadImage(url)
	if err != nil {
		return err
	}
	defer body.Close()
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, body)
	return err
}
