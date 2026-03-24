package main

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

const numWorkers = 10

type threadData struct {
	ID          string
	Builders    []string
	Score       int
	Screenshots []string
	Lore        string
	WhatToVisit string
}

type threadResult struct {
	idx   int
	guild Guild
}

func syncGuilds(s *discordgo.Session, root string, guildBaseShowcaseChannelForumID string) (SyncStats, error) {
	guilds, err := loadGuilds(root)
	if err != nil {
		slog.Warn("loading guilds, starting fresh", "err", err)
		guilds = []Guild{}
	}

	forumChannel, err := s.Channel(guildBaseShowcaseChannelForumID)
	if err != nil {
		return SyncStats{}, fmt.Errorf("fetching channel: %w", err)
	}

	tagMap := make(map[string]string)
	for _, tag := range forumChannel.AvailableTags {
		tagMap[tag.ID] = tag.Name
	}
	slog.Info("forum tags loaded", "count", len(tagMap))

	active, err := s.GuildThreadsActive(forumChannel.GuildID)
	if err != nil {
		return SyncStats{}, fmt.Errorf("fetching active threads: %w", err)
	}

	archived, err := s.ThreadsArchived(guildBaseShowcaseChannelForumID, nil, 0)
	if err != nil {
		slog.Warn("fetching archived threads", "err", err)
	}

	var threads []*discordgo.Channel
	for _, t := range active.Threads {
		if t.ParentID == guildBaseShowcaseChannelForumID {
			threads = append(threads, t)
		}
	}
	if archived != nil {
		threads = append(threads, archived.Threads...)
	}

	var mu sync.Mutex
	var newCount int
	var newNames []string
	var updatedCount int
	var updatedNames []string

	guildMap := make(map[string]int)
	for i := range guilds {
		guildMap[strings.ToLower(guilds[i].Name)] = i
	}
	for _, thread := range threads {
		name := extractGuildName(thread.Name)
		if _, exists := guildMap[strings.ToLower(name)]; !exists {
			guilds = append(guilds, Guild{Name: name, Builders: []string{}})
			guildMap[strings.ToLower(name)] = len(guilds) - 1
			newCount++
			newNames = append(newNames, name)
			slog.Info("new guild detected", "name", name, "thread", thread.Name)
		}
	}

	type work struct {
		thread *discordgo.Channel
		idx    int
	}

	jobs := make(chan work, len(threads))
	results := make(chan threadResult, len(threads))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				data := fetchThreadData(s, j.thread.ID)

				var tags []string
				for _, tagID := range j.thread.AppliedTags {
					if tagName, ok := tagMap[tagID]; ok {
						tags = append(tags, tagName)
					}
				}

				g := guilds[j.idx]
				if g.ID == "" && data.ID != "" {
					g.ID = data.ID
				}
				g.Builders = data.Builders
				g.Tags = tags
				g.DiscordThread = fmt.Sprintf("https://discord.com/channels/%s/%s", j.thread.GuildID, j.thread.ID)
				g.Score = data.Score
				g.Screenshots = data.Screenshots
				g.Lore = data.Lore
				g.WhatToVisit = data.WhatToVisit

				results <- threadResult{idx: j.idx, guild: g}
			}
		}()
	}

	for _, thread := range threads {
		mu.Lock()
		idx := guildMap[strings.ToLower(extractGuildName(thread.Name))]
		mu.Unlock()
		jobs <- work{thread: thread, idx: idx}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		prev := guilds[r.idx]
		changed := prev.Score != r.guild.Score ||
			len(prev.Screenshots) != len(r.guild.Screenshots) ||
			strings.Join(prev.Builders, ",") != strings.Join(r.guild.Builders, ",")
		if changed {
			updatedCount++
			updatedNames = append(updatedNames, r.guild.Name)
		}
		guilds[r.idx] = r.guild
		slog.Info("guild synced",
			"name", r.guild.Name,
			"id", r.guild.ID,
			"score", r.guild.Score,
			"builders", strings.Join(r.guild.Builders, ", "),
			"tags", strings.Join(r.guild.Tags, ", "),
			"screenshots", len(r.guild.Screenshots),
		)
	}

	stats := SyncStats{
		Total:        len(guilds),
		New:          newCount,
		NewNames:     newNames,
		Updated:      updatedCount,
		UpdatedNames: updatedNames,
	}

	if DRY_RUN {
		slog.Info("dry-run: skipping save")
		return stats, nil
	}

	if err := saveGuilds(root, guilds); err != nil {
		return SyncStats{}, fmt.Errorf("saving guilds: %w", err)
	}

	slog.Info("sync complete", "total_guilds", stats.Total, "new", stats.New, "updated", stats.Updated, "threads_processed", len(threads))
	return stats, nil
}

func fetchThreadData(s *discordgo.Session, threadID string) threadData {
	msgs, err := s.ChannelMessages(threadID, 1, "", "0", "")
	if err != nil || len(msgs) == 0 {
		slog.Warn("fetching messages", "thread", threadID, "err", err)
		return threadData{}
	}

	id, builders, lore, whatToVisit := parseFirstPost(msgs[0].Content)

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
		Lore:        lore,
		WhatToVisit: whatToVisit,
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

func formatSyncSummary(s SyncStats) string {
	lines := []string{
		"✨ **All guilds have been synchronized!**",
		fmt.Sprintf("🏰 **%d** guilds tracked", s.Total),
	}
	if s.New > 0 {
		names := strings.Join(s.NewNames, ", ")
		lines = append(lines, fmt.Sprintf("🆕 **%d** new guild(s) discovered: %s", s.New, names))
	}
	if s.Updated > 0 {
		names := strings.Join(s.UpdatedNames, ", ")
		lines = append(lines, fmt.Sprintf("🔄 **%d** guild(s) refreshed: %s", s.Updated, names))
	}
	// if s.New == 0 && s.Updated == 0 {
	// 	lines = append(lines, "💤 Nothing changed")
	// }
	// lines = append(lines, fmt.Sprintf("🕐 %s UTC", time.Now().UTC().Format("Jan 2, 2006 · 15:04")))
	return strings.Join(lines, "\n")
}

func notify(s *discordgo.Session, channelID, msg string) {
	if DO_NOT_NOTIFY {
		slog.Info("Discord notification ignored", "msg", msg)
		return
	}
	if channelID == "" {
		slog.Warn("BOT_CHANNEL_ID not set, skipping notification")
		return
	}
	if _, err := s.ChannelMessageSend(channelID, msg); err != nil {
		slog.Warn("failed to send bot notification", "err", err)
	}

}
