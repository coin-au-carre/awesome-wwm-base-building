package discord

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)


const numWorkers = 20

type SyncStats struct {
	Total            int
	New              int
	Updated          int
	NewNames         []string
	UpdatedNames     []string
	NewThreadLinks   map[string]string // guild name → discord thread URL
	VoterGuildCounts map[string]int    // userID → number of distinct guilds voted on
}

type SyncConfig struct {
	ForumChannelID       string
	DryRun               bool
	// ExternalVoterWeights, when set, replaces the internally computed weights.
	// Pass pre-merged weights from all channels for cross-channel scoring.
	ExternalVoterWeights map[string]int
}

type threadData struct {
	ID          string
	GuildName   string
	AuthorID    string
	Builders    []string
	Score       int
	Screenshots []string
	Videos      []string
	Lore        string
	WhatToVisit string
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

	t0 := time.Now()
	threads, err := collectThreads(b.Session, cfg.ForumChannelID, forumChannel.GuildID)
	if err != nil {
		return nil, SyncStats{}, err
	}
	slog.Info("threads collected", "count", len(threads), "elapsed", time.Since(t0).Round(time.Millisecond))

	var stats SyncStats
	stats.NewThreadLinks = make(map[string]string)
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
			stats.NewThreadLinks[name] = fmt.Sprintf("https://discord.com/channels/%s/%s", thread.GuildID, thread.ID)
			slog.Info("new guild detected", "name", name, "thread", thread.Name)
		}
	}

	type contentWork struct {
		thread *discordgo.Channel
		idx    int
	}
	type contentResult struct {
		idx       int
		thread    *discordgo.Channel
		data      threadData
		reactions map[string][]string // emoji → []userID for this thread
	}

	contentJobs := make(chan contentWork, len(threads))
	contentResults := make(chan contentResult, len(threads))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range contentJobs {
				tThread := time.Now()
				var (
					data      threadData
					reactions map[string][]string
					wg2       sync.WaitGroup
				)
				wg2.Add(2)
				go func() { defer wg2.Done(); data = fetchThreadContent(b.Session, j.thread) }()
				go func() { defer wg2.Done(); reactions = fetchThreadReactions(b.Session, j.thread.ID) }()
				wg2.Wait()
				slog.Info("thread fetched",
					"name", guild.ExtractName(j.thread.Name),
					"id", data.ID,
					"builders", strings.Join(data.Builders, ", "),
					"screenshots", len(data.Screenshots),
					"reactions", totalReactions(reactions),
					"elapsed", time.Since(tThread).Round(time.Millisecond),
				)
				contentResults <- contentResult{
					idx:       j.idx,
					thread:    j.thread,
					data:      data,
					reactions: reactions,
				}
			}
		}()
	}

	for _, thread := range threads {
		idx := guildMap[strings.ToLower(guild.ExtractName(thread.Name))]
		contentJobs <- contentWork{thread: thread, idx: idx}
	}
	close(contentJobs)

	go func() {
		wg.Wait()
		close(contentResults)
	}()

	// Collect all results, then compute voter weights from the global reaction picture.
	tFetch := time.Now()
	rawResults := make([]contentResult, 0, len(threads))
	userGuilds := make(map[string]map[string]bool)
	for r := range contentResults {
		rawResults = append(rawResults, r)
		for _, users := range r.reactions {
			for _, uid := range users {
				if userGuilds[uid] == nil {
					userGuilds[uid] = make(map[string]bool)
				}
				userGuilds[uid][r.thread.ID] = true
			}
		}
	}
	slog.Info("content and reactions fetched", "threads", len(rawResults), "elapsed", time.Since(tFetch).Round(time.Millisecond))

	voterGuildCounts := make(map[string]int, len(userGuilds))
	for uid, guilds := range userGuilds {
		voterGuildCounts[uid] = len(guilds)
	}

	var voterWeights map[string]int
	if len(cfg.ExternalVoterWeights) > 0 {
		voterWeights = cfg.ExternalVoterWeights
		slog.Info("using external voter weights", "voters", len(voterWeights))
	} else {
		voterWeights = ComputeVoterWeights(voterGuildCounts)
		slog.Info("voter weights computed", "voters", len(voterWeights))
	}
	stats.VoterGuildCounts = voterGuildCounts

	for _, r := range rawResults {
		data := r.data
		data.Score = computeScore(r.reactions, voterWeights, data.Lore, data.WhatToVisit)

		var tags []string
		for _, tagID := range r.thread.AppliedTags {
			if name, ok := tagMap[tagID]; ok {
				tags = append(tags, name)
			}
		}

		g := guilds[r.idx]
		if g.ID == "" && data.ID != "" {
			g.ID = data.ID
		}
		if data.GuildName != "" {
			if strings.EqualFold(data.GuildName, g.Name) {
				g.GuildName = "" // parsed name is same as base name, no need for a separate display name
			} else {
				g.GuildName = data.GuildName
			}
		}
		prev := g
		g.Builders = data.Builders
		g.Tags = tags
		g.DiscordThread = fmt.Sprintf("https://discord.com/channels/%s/%s", r.thread.GuildID, r.thread.ID)
		g.Score = data.Score
		g.Screenshots = data.Screenshots
		g.Videos = data.Videos
		g.Lore = data.Lore
		g.WhatToVisit = data.WhatToVisit
		g.BuilderDiscordID = data.AuthorID

		if !newIndices[r.idx] && hasChanged(prev, g) {
			stats.Updated++
			stats.UpdatedNames = append(stats.UpdatedNames, g.Name)
		}

		guilds[r.idx] = g
		slog.Info("guild scored", "name", g.Name, "score", g.Score, "tags", strings.Join(g.Tags, ", "))
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

func fetchThreadContent(s *discordgo.Session, thread *discordgo.Channel) threadData {
	msgs, err := s.ChannelMessages(thread.ID, 1, "", "0", "")
	if err != nil || len(msgs) == 0 {
		slog.Warn("fetching messages", "thread", thread.ID, "err", err)
		return threadData{}
	}

	id, guildName, builders, lore, whatToVisit := guild.ParseFirstPost(msgs[0].Content)
	authorID := msgs[0].Author.ID
	screenshots, videos := collectMedia(s, thread.ID, authorID)

	return threadData{
		ID:          id,
		GuildName:   guildName,
		AuthorID:    authorID,
		Builders:    resolveBuilders(s, builders),
		Screenshots: screenshots,
		Videos:      videos,
		Lore:        lore,
		WhatToVisit: whatToVisit,
	}
}


var reMention = regexp.MustCompile(`^<@!?(\d+)>$`)

func resolveBuilders(s *discordgo.Session, builders []string) []string {
	resolved := make([]string, 0, len(builders))
	for _, b := range builders {
		if m := reMention.FindStringSubmatch(b); len(m) == 2 {
			if u, err := s.User(m[1]); err == nil {
				resolved = append(resolved, u.Username)
				continue
			}
		}
		resolved = append(resolved, b)
	}
	return resolved
}

const maxScreenshots = 40

func collectMedia(s *discordgo.Session, threadID, authorID string) (screenshots, videos []string) {
	seen := make(map[string]bool)
	var lastID string

	for {
		if len(screenshots) >= maxScreenshots {
			break
		}
		msgs, err := s.ChannelMessages(threadID, 100, lastID, "", "")
		if err != nil || len(msgs) == 0 {
			break
		}
		for _, msg := range msgs {
			if msg.Author == nil || msg.Author.ID != authorID {
				continue
			}
			for _, att := range msg.Attachments {
				if seen[att.URL] {
					continue
				}
				seen[att.URL] = true
				if guild.IsImage(att.Filename) {
					screenshots = append(screenshots, att.URL)
					slog.Debug("screenshot found", "thread", threadID, "url", att.URL)
				} else if guild.IsVideo(att.Filename) {
					videos = append(videos, att.URL)
					slog.Debug("video found", "thread", threadID, "url", att.URL)
				}
			}
			for _, embed := range msg.Embeds {
				if embed.Type == discordgo.EmbedTypeVideo && embed.URL != "" && !seen[embed.URL] {
					seen[embed.URL] = true
					videos = append(videos, embed.URL)
					slog.Debug("embed video found", "thread", threadID, "url", embed.URL)
				} else if embed.Image != nil && embed.Image.URL != "" && !seen[embed.Image.URL] {
					seen[embed.Image.URL] = true
					screenshots = append(screenshots, embed.Image.URL)
					slog.Debug("embed image found", "thread", threadID, "url", embed.Image.URL)
				}
			}
		}
		lastID = msgs[len(msgs)-1].ID
		if len(msgs) < 100 {
			break
		}
	}
	return
}

func totalReactions(reactions map[string][]string) int {
	n := 0
	for _, users := range reactions {
		n += len(users)
	}
	return n
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
		len(prev.Videos) != len(next.Videos) ||
		strings.Join(prev.Builders, ",") != strings.Join(next.Builders, ",")
}
