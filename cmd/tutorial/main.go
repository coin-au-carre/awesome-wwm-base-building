// cmd/tutorial/main.go
package main

import (
	"bufio"
	"bytes"
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
	"strings"
	"time"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"
	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

// Captures server/channel/message — prefers the third segment (thread ID) when present.
var reURL = regexp.MustCompile(`discord\.com/channels/\d+/(\d+)(?:/(\d+))?`)
var reSlug = regexp.MustCompile(`[^\p{L}\p{N}]+`)
var reDiscordVideoMarker = regexp.MustCompile(`<!--\s*discord-video:(\d+)/(\d+)\s*-->`)
var reVideoSrc = regexp.MustCompile(`(<video\s[^>]*src=)"[^"]*"`)
var reVideoSrcURL = regexp.MustCompile(`<video\b[^>]*\bsrc="(https://[^"]+)"`)
var reDiscordCDN = regexp.MustCompile(`https://(?:cdn\.discordapp\.com|media\.discordapp\.net)/attachments/[^\s"'<>]+`)

func slugify(s string) string {
	return strings.Trim(reSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func main() {
	root := flag.String("root", cmdutil.RootDir(), "repository root directory")
	list := flag.String("list", "", "file with one Discord thread URL per line")
	refreshVideos := flag.Bool("refresh-videos", false, "refresh Discord CDN video URLs in all articles (requires <!-- discord-video --> markers)")
	refreshImages := flag.Bool("refresh-images", false, "refresh all Discord CDN image/video src URLs in all articles via the Discord refresh-urls API")
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

	// Also collect discordThread URLs from existing article frontmatters.
	articlesDirEarly := filepath.Join(*root, "web", "src", "content", "articles")
	urls = append(urls, collectFrontmatterThreadURLs(articlesDirEarly)...)

	// Deduplicate URLs.
	seen := make(map[string]bool, len(urls))
	var deduped []string
	for _, u := range urls {
		if !seen[u] {
			seen[u] = true
			deduped = append(deduped, u)
		}
	}
	urls = deduped

	if len(urls) == 0 && !*refreshVideos && !*refreshImages {
		fmt.Fprintln(os.Stderr, "usage: tutorial [-list <file>] [-refresh-videos] [-refresh-images] <discord-thread-url>...")
		os.Exit(1)
	}

	cmdutil.LoadEnv(*root)

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

	articlesDir := filepath.Join(*root, "web", "src", "content", "articles")

	if *refreshVideos {
		if err := refreshVideoURLs(bot.Session, articlesDir); err != nil {
			slog.Error("refreshing video URLs", "err", err)
			os.Exit(1)
		}
	}

	if *refreshImages {
		if err := refreshAllCDNURLs(bot.Session.Token, articlesDir); err != nil {
			slog.Error("refreshing image URLs", "err", err)
			os.Exit(1)
		}
	}
}

func syncThread(s *discordgo.Session, root, rawURL string) error {
	m := reURL.FindStringSubmatch(rawURL)
	if m == nil {
		return fmt.Errorf("invalid Discord thread URL: %s", rawURL)
	}
	channelID := m[1]
	messageID := m[2] // non-empty when URL points to a specific message

	thread, err := s.Channel(channelID)
	if err != nil {
		return fmt.Errorf("fetching thread: %w", err)
	}

	var allMsgs []*discordgo.Message
	if messageID != "" {
		// URL points to a specific message — fetch only that message.
		msg, err := s.ChannelMessage(channelID, messageID)
		if err != nil {
			return fmt.Errorf("fetching message: %w", err)
		}
		allMsgs = []*discordgo.Message{msg}
	} else {
		allMsgs = fetchAllMessages(s, channelID)
	}
	if len(allMsgs) == 0 {
		return fmt.Errorf("no messages found in channel %s", channelID)
	}

	authorID := allMsgs[0].Author.ID
	authorName := allMsgs[0].Author.Username
	if mem, err := s.GuildMember(thread.GuildID, authorID); err == nil && mem.Nick != "" {
		authorName = mem.Nick
	}

	articlesDir := filepath.Join(root, "web", "src", "content", "articles")
	slug := slugify(thread.Name)
	var firstImageURL string
	var parts []string

	userMap, _ := guild.LoadUsers(root)

	var groups [][]string
	for _, msg := range allMsgs {
		text := strings.TrimSpace(msg.Content)
		if text != "" {
			text = resolveDiscordMentions(s, userMap, text)
		}
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

	lookupID := channelID
	if messageID != "" {
		lookupID = messageID
	}
	outPath := findArticleByThreadID(articlesDir, lookupID)
	if outPath == "" {
		outPath = filepath.Join(articlesDir, slug+".md")
	}

	var existingFrontmatter string
	var preservedBlocks []string
	if data, err := os.ReadFile(outPath); err == nil {
		s := string(data)
		if fm := extractFrontmatter(s); fm != "" {
			existingFrontmatter = fm
		}
		preservedBlocks = extractPreservedBlocks(s)
		// Fall back to legacy preserve-below marker.
		if len(preservedBlocks) == 0 {
			if tail := extractPreservedTail(s); tail != "" {
				preservedBlocks = []string{tail}
			}
		}
	}

	var content string
	body := strings.Join(parts, "\n\n")
	for _, block := range preservedBlocks {
		body = body + "\n\n<!-- preserve-start -->\n\n" + block + "\n\n<!-- preserve-end -->"
	}
	if existingFrontmatter != "" {
		fm := refreshFrontmatterImage(existingFrontmatter, firstImageURL)
		content = fmt.Sprintf("---\n%s---\n\n%s\n", fm, body)
	} else {
		content = buildMarkdown(thread.Name, authorName, firstImageURL, []string{body})
	}

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing article: %w", err)
	}

	// Persist author's Discord ID to users.json so the credits page can display their name.
	info := guild.UserInfo{
		Username:   allMsgs[0].Author.Username,
		GlobalName: allMsgs[0].Author.GlobalName,
	}
	if mem, err := s.GuildMember(thread.GuildID, authorID); err == nil && mem.Nick != "" {
		info.Nickname = mem.Nick
	}
	userMap[authorID] = info
	if err := guild.SaveUsers(root, userMap); err != nil {
		slog.Warn("saving users.json", "err", err)
	}

	fmt.Printf("wrote  %s\n", outPath)
	fmt.Printf("author %s (%s)\n", authorName, authorID)
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

var reDiscordThreadField = regexp.MustCompile(`(?m)^discordThread:\s*"([^"]+)"`)
var reChannelMention = regexp.MustCompile(`<#(\d+)>`)
var reUserMention = regexp.MustCompile(`<@!?(\d+)>`)

// resolveDiscordMentions replaces <#CHANNEL_ID> and <@USER_ID> with human-readable names.
func resolveDiscordMentions(s *discordgo.Session, userMap map[string]guild.UserInfo, text string) string {
	text = reChannelMention.ReplaceAllStringFunc(text, func(match string) string {
		id := reChannelMention.FindStringSubmatch(match)[1]
		if ch, err := s.Channel(id); err == nil {
			return "#" + ch.Name
		}
		return match
	})
	text = reUserMention.ReplaceAllStringFunc(text, func(match string) string {
		id := reUserMention.FindStringSubmatch(match)[1]
		if info, ok := userMap[id]; ok {
			if info.Nickname != "" {
				return info.Nickname
			}
			if info.GlobalName != "" {
				return info.GlobalName
			}
			if info.Username != "" {
				return info.Username
			}
		}
		return match
	})
	return text
}

// collectFrontmatterThreadURLs scans all articles for a discordThread frontmatter field
// and returns the URLs so they are synced automatically without needing tutorial_threads.txt.
func collectFrontmatterThreadURLs(articlesDir string) []string {
	entries, _ := os.ReadDir(articlesDir)
	var urls []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(articlesDir, e.Name()))
		if err != nil {
			continue
		}
		fm := extractFrontmatter(string(data))
		if m := reDiscordThreadField.FindStringSubmatch(fm); m != nil {
			urls = append(urls, m[1])
		}
	}
	return urls
}

// findArticleByThreadID scans all articles for one whose frontmatter contains
// discordThread: <threadID> and returns its path, or "" if not found.
func findArticleByThreadID(articlesDir, threadID string) string {
	entries, _ := os.ReadDir(articlesDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(articlesDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		fm := extractFrontmatter(string(data))
		if strings.Contains(fm, threadID) {
			return path
		}
	}
	return ""
}

// extractPreservedBlocks returns all content between <!-- preserve-start --> and
// <!-- preserve-end --> tags, in order, as separate strings.
func extractPreservedBlocks(content string) []string {
	const open = "<!-- preserve-start -->"
	const close = "<!-- preserve-end -->"
	var blocks []string
	rest := content
	for {
		start := strings.Index(rest, open)
		if start < 0 {
			break
		}
		rest = rest[start+len(open):]
		end := strings.Index(rest, close)
		if end < 0 {
			break
		}
		block := strings.TrimSpace(rest[:end])
		if block != "" {
			blocks = append(blocks, block)
		}
		rest = rest[end+len(close):]
	}
	return blocks
}

// extractPreservedTail is the legacy single-marker form; kept for back-compat.
func extractPreservedTail(content string) string {
	const marker = "<!-- preserve-below -->"
	idx := strings.Index(content, marker)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(content[idx+len(marker):])
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

func firstVideoAttachment(atts []*discordgo.MessageAttachment) string {
	for _, att := range atts {
		if isVideo(mediaExt(att.URL)) {
			return att.URL
		}
	}
	return ""
}

// fetchSnapshotVideoURL fetches the raw Discord message JSON and looks for a
// video attachment inside message_snapshots (forwarded/transferred messages).
func fetchSnapshotVideoURL(token, channelID, messageID string) string {
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s", channelID, messageID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var raw struct {
		MessageSnapshots []struct {
			Message struct {
				Attachments []struct {
					URL string `json:"url"`
				} `json:"attachments"`
			} `json:"message"`
		} `json:"message_snapshots"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return ""
	}
	for _, snap := range raw.MessageSnapshots {
		for _, att := range snap.Message.Attachments {
			if isVideo(mediaExt(att.URL)) {
				return att.URL
			}
		}
	}
	return ""
}

// toCDNForm normalizes any Discord attachment URL (cdn or media) to a plain
// cdn.discordapp.com URL with only the auth params (ex/is/hm), suitable for
// the Discord refresh-urls API which does not accept media.discordapp.net.
func toCDNForm(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	u.Host = "cdn.discordapp.com"
	q := u.Query()
	newQ := make(url.Values)
	for _, key := range []string{"ex", "is", "hm"} {
		if v := q.Get(key); v != "" {
			newQ.Set(key, v)
		}
	}
	u.RawQuery = newQ.Encode()
	return u.String(), nil
}

// toMediaForm converts a cdn.discordapp.com URL to media.discordapp.net with
// WebP encoding, which is lighter and faster for browser display.
func toMediaForm(cdnURL string) string {
	u, err := url.Parse(cdnURL)
	if err != nil {
		return cdnURL
	}
	u.Host = "media.discordapp.net"
	q := u.Query()
	q.Set("format", "webp")
	q.Set("quality", "lossless")
	u.RawQuery = q.Encode()
	return u.String()
}

// refreshAllCDNURLs scans all articles for Discord CDN URLs and refreshes their
// tokens via the Discord refresh-urls API, without touching any other content.
func refreshAllCDNURLs(token, articlesDir string) error {
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
		origURLs := reDiscordCDN.FindAllString(content, -1)
		if len(origURLs) == 0 {
			continue
		}
		// Collect video src URLs — these must stay as cdn.discordapp.com and must
		// not be converted to media.discordapp.net (videos aren't images).
		videoURLs := make(map[string]bool)
		for _, m := range reVideoSrcURL.FindAllStringSubmatch(content, -1) {
			videoURLs[m[1]] = true
		}
		// Normalize all URLs to cdn form for the API (media.discordapp.net not accepted).
		// Build a deduped list and a mapping: cdn_form → original URL.
		cdnForms := make(map[string]string) // cdn_form → first original that maps to it
		var uniqueCDN []string
		for _, orig := range origURLs {
			cdn, err := toCDNForm(orig)
			if err != nil {
				continue
			}
			if _, seen := cdnForms[cdn]; !seen {
				cdnForms[cdn] = orig
				uniqueCDN = append(uniqueCDN, cdn)
			}
		}
		refreshed, err := bulkRefreshDiscordURLs(token, uniqueCDN)
		if err != nil {
			slog.Warn("refreshing CDN URLs", "file", e.Name(), "err", err)
			continue
		}
		// Replace each original URL with the refreshed URL.
		// Image URLs are converted to media.discordapp.net (WebP); video URLs stay as cdn.
		updated := content
		changed := false
		for _, orig := range origURLs {
			cdn, err := toCDNForm(orig)
			if err != nil {
				continue
			}
			newCDN, ok := refreshed[cdn]
			if !ok {
				continue
			}
			var newURL string
			if videoURLs[orig] {
				newURL = newCDN
			} else {
				newURL = toMediaForm(newCDN)
			}
			if newURL != orig {
				updated = strings.ReplaceAll(updated, orig, newURL)
				changed = true
			}
		}
		if !changed {
			continue
		}
		if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
			slog.Warn("writing article", "file", e.Name(), "err", err)
			continue
		}
		slog.Info("refreshed CDN URLs", "file", e.Name(), "count", len(refreshed))
	}
	return nil
}

func bulkRefreshDiscordURLs(token string, urls []string) (map[string]string, error) {
	body, err := json.Marshal(map[string]any{"attachment_urls": urls})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", "https://discord.com/api/v10/attachments/refresh-urls", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord API %d: %s", resp.StatusCode, b)
	}
	var result struct {
		RefreshedURLs []struct {
			Original  string `json:"original"`
			Refreshed string `json:"refreshed"`
		} `json:"refreshed_urls"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	m := make(map[string]string, len(result.RefreshedURLs))
	for _, r := range result.RefreshedURLs {
		m[r.Original] = r.Refreshed
	}
	return m, nil
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
		cdnURL := firstVideoAttachment(msg.Attachments)
		if cdnURL == "" {
			// Forwarded messages store attachments in message_snapshots — discordgo
			// doesn't parse that field, so fall back to a raw REST call.
			cdnURL = fetchSnapshotVideoURL(s.Token, channelID, messageID)
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

