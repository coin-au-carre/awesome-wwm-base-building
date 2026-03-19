package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type Guild struct {
	ID            string   `json:"id,omitempty"`
	Name          string   `json:"name"`
	Builders      []string `json:"builders"`
	Tags          []string `json:"tags,omitempty"`
	DiscordThread string   `json:"discordThread"`
	Score         int      `json:"score"`
	Screenshots   []string `json:"screenshots,omitempty"`
}

type threadData struct {
	ID          string
	Builders    []string
	Score       int
	Screenshots []string
}

var (
	reBracketID = regexp.MustCompile(`\[(\d+)\]`)
	reBuilders  = regexp.MustCompile(`(?i)builders?:\s*(.+)`)
)

const (
	GUILD_BUILDING_CHANNEL_FORUM_ID = "1483455027250200639"
	DRY_RUN                         = false
)

// rootDir returns the repo root, whether ruby.go is run from / or ruby/
func rootDir() string {
	if _, err := os.Stat("LICENSE"); err == nil {
		return "."
	}
	return ".."
}

func main() {
	root := rootDir()
	if err := godotenv.Load(filepath.Join(root, ".env")); err != nil {
		slog.Error("loading .env", "err", err)
		os.Exit(1)
	}

	session, err := discordgo.New("Bot " + os.Getenv("RUBY_BOT_TOKEN"))
	if err != nil {
		slog.Error("creating session", "err", err)
		os.Exit(1)
	}
	defer session.Close()

	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent

	if err := session.Open(); err != nil {
		slog.Error("opening session", "err", err)
		os.Exit(1)
	}

	if err := sync(session, root); err != nil {
		slog.Error("sync failed", "err", err)
		os.Exit(1)
	}
}

func sync(s *discordgo.Session, root string) error {
	guilds, err := loadGuilds(root)
	if err != nil {
		slog.Warn("loading guilds, starting fresh", "err", err)
		guilds = []Guild{}
	}

	forumChannel, err := s.Channel(GUILD_BUILDING_CHANNEL_FORUM_ID)
	if err != nil {
		return fmt.Errorf("fetching channel: %w", err)
	}

	tagMap := make(map[string]string)
	for _, tag := range forumChannel.AvailableTags {
		tagMap[tag.ID] = tag.Name
	}
	slog.Info("forum tags loaded", "count", len(tagMap))

	active, err := s.GuildThreadsActive(forumChannel.GuildID)
	if err != nil {
		return fmt.Errorf("fetching active threads: %w", err)
	}

	archived, err := s.ThreadsArchived(GUILD_BUILDING_CHANNEL_FORUM_ID, nil, 0)
	if err != nil {
		slog.Warn("fetching archived threads", "err", err)
	}

	var threads []*discordgo.Channel
	for _, t := range active.Threads {
		if t.ParentID == GUILD_BUILDING_CHANNEL_FORUM_ID {
			threads = append(threads, t)
		}
	}
	if archived != nil {
		threads = append(threads, archived.Threads...)
	}

	guildMap := make(map[string]int)
	for i := range guilds {
		guildMap[strings.ToLower(guilds[i].Name)] = i
	}

	for _, thread := range threads {
		name := extractGuildName(thread.Name)
		idx, exists := guildMap[strings.ToLower(name)]
		if !exists {
			guilds = append(guilds, Guild{Name: name, Builders: []string{}})
			idx = len(guilds) - 1
			guildMap[strings.ToLower(name)] = idx
			slog.Info("new guild detected", "name", name, "thread", thread.Name)
		}

		data := fetchThreadData(s, thread.ID)

		if guilds[idx].ID == "" && data.ID != "" {
			guilds[idx].ID = data.ID
		}
		if len(guilds[idx].Builders) == 0 && len(data.Builders) > 0 {
			guilds[idx].Builders = data.Builders
		}

		var tags []string
		for _, tagID := range thread.AppliedTags {
			if tagName, ok := tagMap[tagID]; ok {
				tags = append(tags, tagName)
			}
		}
		guilds[idx].Tags = tags
		guilds[idx].DiscordThread = fmt.Sprintf("https://discord.com/channels/%s/%s", thread.GuildID, thread.ID)
		guilds[idx].Score = data.Score
		guilds[idx].Screenshots = data.Screenshots

		slog.Info("guild synced",
			"name", guilds[idx].Name,
			"id", guilds[idx].ID,
			"score", guilds[idx].Score,
			"builders", strings.Join(guilds[idx].Builders, ", "),
			"tags", strings.Join(guilds[idx].Tags, ", "),
			"screenshots", len(guilds[idx].Screenshots),
		)
	}

	if DRY_RUN {
		slog.Info("dry-run: skipping save")
		return nil
	}

	if err := saveGuilds(root, guilds); err != nil {
		return fmt.Errorf("saving guilds: %w", err)
	}

	slog.Info("sync complete", "total_guilds", len(guilds), "threads_processed", len(threads))
	return nil
}

func fetchThreadData(s *discordgo.Session, threadID string) threadData {
	msgs, err := s.ChannelMessages(threadID, 1, "", "0", "")
	if err != nil || len(msgs) == 0 {
		slog.Warn("fetching messages", "thread", threadID, "err", err)
		return threadData{}
	}

	id, builders := parseFirstPost(msgs[0].Content)

	score := 0
	for _, r := range msgs[0].Reactions {
		pts := 0
		switch r.Emoji.Name {
		case "⭐":
			pts = r.Count * 2
		case "👍":
			pts = r.Count
		case "🔥":
			pts = r.Count
		}
		if pts > 0 {
			slog.Debug("reaction", "thread", threadID, "emoji", r.Emoji.Name, "count", r.Count, "pts", pts)
		}
		score += pts
	}

	return threadData{
		ID:          id,
		Builders:    builders,
		Score:       score,
		Screenshots: collectScreenshotURLs(s, threadID),
	}
}

func collectScreenshotURLs(s *discordgo.Session, threadID string) []string {
	seen := make(map[string]bool)
	var urls []string
	var lastID string

	for {
		msgs, err := s.ChannelMessages(threadID, 100, lastID, "", "")
		if err != nil || len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
			for _, att := range msg.Attachments {
				if isImage(att.Filename) && !seen[att.URL] {
					seen[att.URL] = true
					urls = append(urls, att.URL)
					slog.Debug("screenshot found", "thread", threadID, "url", att.URL)
				}
			}
			for _, embed := range msg.Embeds {
				if embed.Image != nil && embed.Image.URL != "" && !seen[embed.Image.URL] {
					seen[embed.Image.URL] = true
					urls = append(urls, embed.Image.URL)
					slog.Debug("embed image found", "thread", threadID, "url", embed.Image.URL)
				}
			}
		}

		lastID = msgs[len(msgs)-1].ID
		if len(msgs) < 100 {
			break
		}
	}

	return urls
}

func parseFirstPost(content string) (id string, builders []string) {
	if m := reBracketID.FindStringSubmatch(content); len(m) > 1 {
		id = m[1]
	}
	if m := reBuilders.FindStringSubmatch(content); len(m) > 1 {
		for _, b := range strings.Split(m[1], ",") {
			if b = strings.TrimSpace(b); b != "" {
				builders = append(builders, b)
			}
		}
	}
	return
}

func extractGuildName(threadName string) string {
	parts := strings.SplitN(threadName, " -", 2)
	return strings.TrimSpace(strings.Trim(parts[0], "[]🏯📍"))
}

func isImage(filename string) bool {
	switch strings.ToLower(getExt(filename)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}
	return false
}

func getExt(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}

func loadGuilds(root string) ([]Guild, error) {
	data, err := os.ReadFile(filepath.Join(root, "guilds.json"))
	if err != nil {
		return nil, err
	}
	var guilds []Guild
	return guilds, json.Unmarshal(data, &guilds)
}

func saveGuilds(root string, guilds []Guild) error {
	data, err := json.MarshalIndent(guilds, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "guilds.json"), data, 0644)
}
