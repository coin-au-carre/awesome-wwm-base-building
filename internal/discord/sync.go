package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

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
	Total            int
	New              int
	Updated          int
	NewNames         []string
	UpdatedNames     []string
	VoterGuildCounts map[string]int // userID → number of distinct guilds voted on
}

type SyncConfig struct {
	ForumChannelID string
	DryRun         bool
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

	voterWeights := make(map[string]int, len(userGuilds))
	for uid, count := range voterGuildCounts {
		if w := voterWeight(count); w > 0 {
			voterWeights[uid] = w
		}
	}
	slog.Info("voter weights computed", "voters", len(voterWeights))
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
		if data.GuildName != "" && !strings.EqualFold(data.GuildName, g.Name) {
			g.GuildName = data.GuildName
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
		slog.Info("guild synced",
			"name", g.Name,
			"id", g.ID,
			"score", g.Score,
			"builders", strings.Join(g.Builders, ", "),
			"tags", strings.Join(g.Tags, ", "),
			"screenshots", len(g.Screenshots),
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
		Builders:    builders,
		Screenshots: screenshots,
		Videos:      videos,
		Lore:        lore,
		WhatToVisit: whatToVisit,
	}
}

func computeScore(reactions map[string][]string, weights map[string]int, lore, whatToVisit string) int {
	score := 0
	for emoji, users := range reactions {
		pts := 0
		switch emoji {
		case "⭐":
			pts = scorePerStar
		case "👍", "👍🏻", "👍🏼", "👍🏽", "👍🏾", "👍🏿", "🔥":
			pts = scorePerLike
		}
		for _, uid := range users {
			score += pts * weights[uid]
		}
	}
	if lore != "" {
		score += scoreLoreBonus
	}
	if whatToVisit != "" {
		score += scoreVisitBonus
	}
	return score
}

// voterWeight returns the reaction weight for a user based on how many distinct
// guilds they reacted to: 0 if <4, 1 if 4–7, 2 if 8+.
func voterWeight(distinctGuilds int) int {
	switch {
	case distinctGuilds >= 9:
		return 3
	case distinctGuilds >= 6:
		return 2
	case distinctGuilds >= 2:
		return 1
	default:
		return 0
	}
}

var scoredEmojis = []string{
	"⭐",
	"👍", "👍🏻", "👍🏼", "👍🏽", "👍🏾", "👍🏿",
	"🔥",
}

// fetchThreadReactions fetches all reactor user IDs for each scored emoji in a single thread.
// Returns emoji → []userID.
func fetchThreadReactions(s *discordgo.Session, threadID string) map[string][]string {
	reactions := make(map[string][]string)
	for _, emoji := range scoredEmojis {
		var ids []string
		var after string
		for {
			page, err := s.MessageReactions(threadID, threadID, emoji, 100, "", after)
			if err != nil || len(page) == 0 {
				break
			}
			for _, u := range page {
				ids = append(ids, u.ID)
			}
			after = page[len(page)-1].ID
			if len(page) < 100 {
				break
			}
		}
		if len(ids) > 0 {
			reactions[emoji] = ids
		}
	}
	return reactions
}

func collectMedia(s *discordgo.Session, threadID, authorID string) (screenshots, videos []string) {
	seen := make(map[string]bool)
	var lastID string

	for {
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
