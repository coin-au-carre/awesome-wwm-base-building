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
	Videos      []string
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

	slog.Info("collecting reaction data for voter weights", "threads", len(threads))
	threadReactions, voterWeights := collectReactions(b.Session, threads)
	slog.Info("voter weights computed", "voters", len(voterWeights))

	type work struct {
		thread    *discordgo.Channel
		idx       int
		reactions map[string][]string // emoji → []userID, pre-fetched
	}

	jobs := make(chan work, len(threads))
	results := make(chan threadResult, len(threads))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				data := fetchThreadData(b.Session, j.thread, j.reactions, voterWeights)

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
				g.Videos = data.Videos
				g.Lore = data.Lore
				g.WhatToVisit = data.WhatToVisit

				results <- threadResult{idx: j.idx, guild: g, authorID: data.AuthorID}
			}
		}()
	}

	for _, thread := range threads {
		idx := guildMap[strings.ToLower(guild.ExtractName(thread.Name))]
		jobs <- work{thread: thread, idx: idx, reactions: threadReactions[thread.ID]}
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

func fetchThreadData(s *discordgo.Session, thread *discordgo.Channel, reactions map[string][]string, weights map[string]int) threadData {
	msgs, err := s.ChannelMessages(thread.ID, 1, "", "0", "")
	if err != nil || len(msgs) == 0 {
		slog.Warn("fetching messages", "thread", thread.ID, "err", err)
		return threadData{}
	}

	id, guildName, builders, lore, whatToVisit := guild.ParseFirstPost(msgs[0].Content)

	score := 0
	for emoji, users := range reactions {
		pts := 0
		switch emoji {
		case "⭐":
			pts = scorePerStar
		case "👍", "🔥":
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

	slog.Debug("score calculated",
		"thread", thread.Name,
		"weighted_score", score,
		"lore_bonus", lore != "",
		"visit_bonus", whatToVisit != "",
	)

	authorID := msgs[0].Author.ID
	screenshots, videos := collectMedia(s, thread.ID, authorID)

	return threadData{
		ID:          id,
		GuildName:   guildName,
		AuthorID:    authorID,
		Builders:    builders,
		Score:       score,
		Screenshots: screenshots,
		Videos:      videos,
		Lore:        lore,
		WhatToVisit: whatToVisit,
	}
}

// voterWeight returns the reaction weight for a user based on how many distinct
// guilds they reacted to: 0 if <4, 1 if 4–7, 2 if 8+.
func voterWeight(distinctGuilds int) int {
	switch {
	case distinctGuilds >= 8:
		return 2
	case distinctGuilds >= 4:
		return 1
	default:
		return 0
	}
}

var scoredEmojis = []string{"⭐", "👍", "🔥"}

const numReactionWorkers = 20

type reactionJob struct {
	threadID string
	emoji    string
}

type reactionResult struct {
	threadID string
	emoji    string
	userIDs  []string
}

// collectReactions fetches all reactor user IDs for each scored emoji across
// all threads in parallel. Returns:
//   - threadReactions: threadID → emoji → []userID
//   - voterWeights:    userID → weight (based on breadth of voting)
func collectReactions(s *discordgo.Session, threads []*discordgo.Channel) (map[string]map[string][]string, map[string]int) {
	jobs := make(chan reactionJob, len(threads)*len(scoredEmojis))
	results := make(chan reactionResult, len(threads)*len(scoredEmojis))

	var wg sync.WaitGroup
	for range numReactionWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				var ids []string
				var after string
				for {
					page, err := s.MessageReactions(j.threadID, j.threadID, j.emoji, 100, "", after)
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
				results <- reactionResult{threadID: j.threadID, emoji: j.emoji, userIDs: ids}
			}
		}()
	}

	for _, thread := range threads {
		for _, emoji := range scoredEmojis {
			jobs <- reactionJob{threadID: thread.ID, emoji: emoji}
		}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	threadReactions := make(map[string]map[string][]string, len(threads))
	userGuilds := make(map[string]map[string]bool)

	for r := range results {
		if len(r.userIDs) == 0 {
			continue
		}
		if threadReactions[r.threadID] == nil {
			threadReactions[r.threadID] = make(map[string][]string)
		}
		threadReactions[r.threadID][r.emoji] = r.userIDs
		for _, uid := range r.userIDs {
			if userGuilds[uid] == nil {
				userGuilds[uid] = make(map[string]bool)
			}
			userGuilds[uid][r.threadID] = true
		}
	}

	weights := make(map[string]int, len(userGuilds))
	for uid, guilds := range userGuilds {
		w := voterWeight(len(guilds))
		if w > 0 {
			weights[uid] = w
		}
		slog.Debug("voter weight", "user", uid, "distinct_guilds", len(guilds), "weight", w)
	}
	return threadReactions, weights
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
