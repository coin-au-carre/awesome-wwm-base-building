package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

const (
	scorePerStar    = 2
	scorePerLike    = 1
	scorePerFire    = 1
	scoreLoreBonus  = 1
	scoreVisitBonus = 1
)

const numWorkers = 10

type SyncStats struct {
	Total        int
	New          int
	Updated      int
	NewNames     []string
	UpdatedNames []string
}

type SyncConfig struct {
	ForumChannelID    string
	BaseBuilderRoleID string
	DryRun            bool
	ForceRoleAssign   bool
}

type threadData struct {
	ID          string
	GuildName   string
	AuthorID    string
	Builders    []string
	Score       int
	Screenshots []string
	Lore        string
	WhatToVisit string
}

type threadResult struct {
	idx      int
	guild    guild.Guild
	authorID string
}

func Sync(b *Bot, guilds []guild.Guild, cfg SyncConfig) ([]guild.Guild, SyncStats, error) {
	forumChannel, err := b.Session.Channel(cfg.ForumChannelID)
	if err != nil {
		return nil, SyncStats{}, fmt.Errorf("fetching channel: %w", err)
	}

	tagMap := make(map[string]string, len(forumChannel.AvailableTags))
	for _, tag := range forumChannel.AvailableTags {
		tagMap[tag.ID] = tag.Name
	}
	slog.Info("forum tags loaded", "count", len(tagMap))

	threads, err := collectThreads(b.Session, cfg.ForumChannelID, forumChannel.GuildID)
	if err != nil {
		return nil, SyncStats{}, err
	}

	var stats SyncStats
	guildMap := buildGuildMap(guilds)
	newIndices := make(map[int]bool) // track which slice indices are brand-new so we don't also flag them as updated

	for _, thread := range threads {
		name := guild.ExtractName(thread.Name)
		key := strings.ToLower(name)
		if _, exists := guildMap[key]; !exists {
			idx := len(guilds)
			guilds = append(guilds, guild.Guild{Name: name, Builders: []string{}})
			guildMap[key] = len(guilds) - 1
			newIndices[idx] = true //
			stats.New++
			stats.NewNames = append(stats.NewNames, name)
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
				data := fetchThreadData(b.Session, j.thread)

				var tags []string
				for _, tagID := range j.thread.AppliedTags {
					if name, ok := tagMap[tagID]; ok {
						tags = append(tags, name)
					}
				}

				g := guilds[j.idx]
				if g.ID == "" && data.ID != "" {
					g.ID = data.ID
				}
				if data.GuildName != "" && !strings.EqualFold(data.GuildName, g.Name) {
					g.GuildName = data.GuildName
				}
				g.Builders = data.Builders
				g.Tags = tags
				g.DiscordThread = fmt.Sprintf("https://discord.com/channels/%s/%s", j.thread.GuildID, j.thread.ID)
				g.Score = data.Score
				g.Screenshots = data.Screenshots
				g.Lore = data.Lore
				g.WhatToVisit = data.WhatToVisit

				results <- threadResult{idx: j.idx, guild: g, authorID: data.AuthorID}
			}
		}()
	}

	for _, thread := range threads {
		idx := guildMap[strings.ToLower(guild.ExtractName(thread.Name))]
		jobs <- work{thread: thread, idx: idx}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	// build set of users who already have the role — skipped when forcing
	assignedUsers := make(map[string]bool)
	if !cfg.ForceRoleAssign {
		for _, g := range guilds {
			if g.BuilderDiscordID != "" {
				assignedUsers[g.BuilderDiscordID] = true
			}
		}
	}

	for r := range results {
		prev := guilds[r.idx]

		if !newIndices[r.idx] && hasChanged(prev, r.guild) {
			stats.Updated++
			stats.UpdatedNames = append(stats.UpdatedNames, r.guild.Name)
		}

		// assign role once per user across all guilds, safe because results loop is sequential
		if !cfg.DryRun && cfg.BaseBuilderRoleID != "" && r.authorID != "" {
			if !assignedUsers[r.authorID] {
				AssignBaseBuilderRole(b.Session, forumChannel.GuildID, r.authorID, cfg.BaseBuilderRoleID)
				assignedUsers[r.authorID] = true
			}
		}
		r.guild.BuilderDiscordID = r.authorID

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

	stats.Total = len(guilds)
	return guilds, stats, nil
}

func collectThreads(s *discordgo.Session, forumChannelID, guildID string) ([]*discordgo.Channel, error) {
	active, err := s.GuildThreadsActive(guildID)
	if err != nil {
		return nil, fmt.Errorf("fetching active threads: %w", err)
	}

	archived, err := s.ThreadsArchived(forumChannelID, nil, 0)
	if err != nil {
		slog.Warn("fetching archived threads", "err", err)
	}

	var threads []*discordgo.Channel
	for _, t := range active.Threads {
		if t.ParentID == forumChannelID {
			threads = append(threads, t)
		}
	}
	if archived != nil {
		threads = append(threads, archived.Threads...)
	}
	return threads, nil
}

func fetchThreadData(s *discordgo.Session, thread *discordgo.Channel) threadData {
	msgs, err := s.ChannelMessages(thread.ID, 1, "", "0", "")
	if err != nil || len(msgs) == 0 {
		slog.Warn("fetching messages", "thread", thread.ID, "err", err)
		return threadData{}
	}

	id, guildName, builders, lore, whatToVisit := guild.ParseFirstPost(msgs[0].Content)

	score := 0
	for _, r := range msgs[0].Reactions {
		switch r.Emoji.Name {
		case "⭐":
			score += r.Count * scorePerStar
		case "👍", "🔥":
			score += r.Count * scorePerLike
		}
	}

	if lore != "" {
		score += scoreLoreBonus
	}
	if whatToVisit != "" {
		score += scoreVisitBonus
	}

	slog.Debug("score calculated",
		"thread", thread.Name,
		"reactions", score,
		"lore_bonus", lore != "",
		"visit_bonus", whatToVisit != "",
	)

	return threadData{
		ID:          id,
		GuildName:   guildName,
		AuthorID:    msgs[0].Author.ID,
		Builders:    builders,
		Score:       score,
		Screenshots: collectScreenshots(s, thread.ID),
		Lore:        lore,
		WhatToVisit: whatToVisit,
	}
}

func collectScreenshots(s *discordgo.Session, threadID string) []string {
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
				if guild.IsImage(att.Filename) && !seen[att.URL] {
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

func buildGuildMap(guilds []guild.Guild) map[string]int {
	m := make(map[string]int, len(guilds))
	for i, g := range guilds {
		m[strings.ToLower(g.Name)] = i
	}
	return m
}

func hasChanged(prev, next guild.Guild) bool {
	return prev.Score != next.Score ||
		len(prev.Screenshots) != len(next.Screenshots) ||
		strings.Join(prev.Builders, ",") != strings.Join(next.Builders, ",")
}
