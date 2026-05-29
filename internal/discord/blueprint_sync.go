package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"ruby/internal/blueprint"
	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

// BlueprintSyncFetchResult holds raw data from a blueprint forum channel fetch.
type BlueprintSyncFetchResult struct {
	Blueprints  []blueprint.Blueprint
	VoterCounts map[string]int
	Users       guild.UserMap
	threads     []fetchedThread
	newIndices  map[int]bool
	tagMap      map[string]string
	Stats       SyncStats
}

// BlueprintSyncFetch collects all threads from a blueprint forum channel and fetches their content.
func BlueprintSyncFetch(b *Bot, blueprints []blueprint.Blueprint, cfg SyncConfig) (BlueprintSyncFetchResult, error) {
	forumChannel, err := b.Session.Channel(cfg.ForumChannelID)
	if err != nil {
		return BlueprintSyncFetchResult{}, fmt.Errorf("fetching channel: %w", err)
	}

	tagMap := make(map[string]string, len(forumChannel.AvailableTags))
	for _, tag := range forumChannel.AvailableTags {
		tagMap[tag.ID] = tag.Name
	}
	slog.Info("blueprint forum tags loaded", "count", len(tagMap))

	t0 := time.Now()
	threads, err := collectThreads(b.Session, cfg.ForumChannelID, forumChannel.GuildID)
	if err != nil {
		return BlueprintSyncFetchResult{}, err
	}
	slog.Info("blueprint threads collected", "count", len(threads), "elapsed", time.Since(t0).Round(time.Millisecond))

	if cfg.GuildFilter != "" {
		filter := strings.ToLower(cfg.GuildFilter)
		filtered := threads[:0]
		for _, t := range threads {
			if strings.Contains(strings.ToLower(t.Name), filter) {
				filtered = append(filtered, t)
			}
		}
		threads = filtered
	}

	var partialStats SyncStats
	partialStats.NewThreadLinks = make(map[string]string)

	// Build URL → index map for existing blueprints.
	threadURLToIdx := make(map[string]int, len(blueprints))
	for i, bp := range blueprints {
		if bp.DiscordThread != "" {
			threadURLToIdx[bp.DiscordThread] = i
		}
	}
	newIndices := make(map[int]bool)

	for _, thread := range threads {
		threadURL := fmt.Sprintf("https://discord.com/channels/%s/%s", thread.GuildID, thread.ID)
		if _, exists := threadURLToIdx[threadURL]; exists {
			continue // already known
		}
		name := guild.ExtractName(thread.Name)
		idx := len(blueprints)
		blueprints = append(blueprints, blueprint.Blueprint{Name: name, DiscordThread: threadURL})
		threadURLToIdx[threadURL] = idx
		newIndices[idx] = true
		partialStats.New++
		partialStats.NewNames = append(partialStats.NewNames, name)
		partialStats.NewThreadLinks[name] = threadURL
		for _, emoji := range []string{"👍", "🔥", "❤️", "⭐"} {
			if err := b.Session.MessageReactionAdd(thread.ID, thread.ID, emoji); err != nil {
				slog.Warn("adding reaction to new blueprint thread", "thread", name, "emoji", emoji, "err", err)
			}
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
		reactions map[string][]string
	}

	contentJobs := make(chan contentWork, len(threads))
	contentResults := make(chan contentResult, len(threads))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range contentJobs {
				var (
					data      threadData
					reactions map[string][]string
					wg2       sync.WaitGroup
				)
				wg2.Add(2)
				go func() {
					defer wg2.Done()
					data = fetchBlueprintContent(b.Session, j.thread)
				}()
				go func() {
					defer wg2.Done()
					reactions = fetchThreadReactions(b.Session, j.thread.ID)
				}()
				wg2.Wait()
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
		link := fmt.Sprintf("https://discord.com/channels/%s/%s", thread.GuildID, thread.ID)
		idx, ok := threadURLToIdx[link]
		if !ok {
			continue
		}
		contentJobs <- contentWork{thread: thread, idx: idx}
	}
	close(contentJobs)

	go func() {
		wg.Wait()
		close(contentResults)
	}()

	tFetch := time.Now()
	var fetched []fetchedThread
	userThreads := make(map[string]map[string]bool)
	for r := range contentResults {
		fetched = append(fetched, fetchedThread{
			idx:       r.idx,
			thread:    r.thread,
			data:      r.data,
			reactions: r.reactions,
		})
		if r.data.AuthorID != "" {
			if userThreads[r.data.AuthorID] == nil {
				userThreads[r.data.AuthorID] = make(map[string]bool)
			}
			userThreads[r.data.AuthorID][r.thread.ID] = true
		}
		for _, users := range r.reactions {
			for _, uid := range users {
				if userThreads[uid] == nil {
					userThreads[uid] = make(map[string]bool)
				}
				userThreads[uid][r.thread.ID] = true
			}
		}
	}
	slog.Info("blueprint content and reactions fetched", "threads", len(fetched), "elapsed", time.Since(tFetch).Round(time.Millisecond))

	voterCounts := make(map[string]int, len(userThreads))
	for uid, threadSet := range userThreads {
		voterCounts[uid] = len(threadSet)
	}

	userCache := resolveUsers(b.Session, guildID(fetched), userThreads)

	return BlueprintSyncFetchResult{
		Blueprints:  blueprints,
		VoterCounts: voterCounts,
		Users:       userCache,
		threads:     fetched,
		newIndices:  newIndices,
		tagMap:      tagMap,
		Stats:       partialStats,
	}, nil
}

// BlueprintSyncFinalize scores blueprints and returns the final list plus stats.
func BlueprintSyncFinalize(result BlueprintSyncFetchResult, voterWeights map[string]float64, blacklist map[string]bool) ([]blueprint.Blueprint, SyncStats) {
	blueprints := result.Blueprints
	stats := result.Stats
	now := guild.ModifiedNow()

	for _, r := range result.threads {
		rxn := filterReactions(r.reactions, blacklist)
		score := computeScore(rxn, voterWeights, nil, "", "")

		var tags []string
		for _, tagID := range r.thread.AppliedTags {
			if name, ok := result.tagMap[tagID]; ok {
				tags = append(tags, name)
			}
		}

		bp := blueprints[r.idx]
		prev := bp

		// Parse first-post fields
		parsed := r.data // threadData already has first post parsed by fetchBlueprintContent

		name := guild.ExtractName(r.thread.Name)
		if name != "" {
			bp.Name = name
		}
		bp.Tags = tags
		bp.Score = score
		bp.Screenshots = r.data.Screenshots
		var labeledSections []guild.ScreenshotSection
		for _, s := range r.data.ScreenshotSections {
			if s.Label != "" {
				labeledSections = append(labeledSections, s)
			}
		}
		bp.ScreenshotSections = labeledSections
		bp.Videos = r.data.Videos
		if idx := parsed.CoverIdx; idx >= 1 && idx <= len(r.data.Screenshots) {
			bp.CoverImage = r.data.Screenshots[idx-1]
		} else {
			bp.CoverImage = ""
		}
		if bp.CreatedAt == "" {
			bp.CreatedAt = r.data.FirstPostTime.UTC().Format(guild.ModifiedLayout)
		}

		// Blueprint-specific fields come from the blueprint parse stored in Lore/WhatToVisit fields
		// (we reuse threadData.Lore for materials and WhatToVisit for price, see fetchBlueprintContent)
		if r.data.Lore != "" {
			bp.Materials = r.data.Lore
		}
		if r.data.GuildName != "" {
			bp.Description = r.data.GuildName
		}
		if r.data.WhatToVisit != "" {
			bp.Price = r.data.WhatToVisit
			lower := strings.ToLower(bp.Price)
			bp.IsFree = strings.Contains(lower, "free")
			bp.IsPayToBuild = lower != "free" && lower != ""
		} else {
			bp.IsFree = true // no price → free by default
			bp.IsPayToBuild = false
		}
		if r.data.BuildTitle != "" {
			bp.BuilderName = r.data.BuildTitle
		} else if bp.BuilderName == "" && r.data.AuthorID != "" {
			if u, ok := result.Users[r.data.AuthorID]; ok {
				bp.BuilderName = u.DisplayName()
			}
		}
		if r.data.ID != "" {
			bp.BuilderID = r.data.ID
		}

		isNew := result.newIndices[r.idx]
		if !isNew && blueprintHasChanged(prev, bp) {
			stats.Updated++
			stats.UpdatedNames = append(stats.UpdatedNames, bp.Name)
			if len(bp.Screenshots) > len(prev.Screenshots) && !screenshotOnCooldown(prev.LastScreenshotNotifiedAt) {
				stats.MoreScreenshotNames = append(stats.MoreScreenshotNames, bp.Name)
				bp.LastScreenshotNotifiedAt = now
				bp.LastModified = now
			}
			if len(bp.Videos) > len(prev.Videos) && !screenshotOnCooldown(prev.LastVideoNotifiedAt) {
				stats.MoreVideoNames = append(stats.MoreVideoNames, bp.Name)
				bp.LastVideoNotifiedAt = now
				bp.LastModified = now
			}
		} else if isNew {
			bp.LastModified = now
		}

		blueprints[r.idx] = bp
		slog.Info("blueprint scored", "name", bp.Name, "score", bp.Score, "paytobuild", bp.IsPayToBuild)
	}

	stats.Total = len(blueprints)
	return blueprints, stats
}

// fetchBlueprintContent fetches the first post and media for a blueprint thread.
// Fields are stored in threadData with repurposed fields:
//   - Lore → Materials
//   - WhatToVisit → Price
//   - BuildTitle → BuilderName
//   - ID → BuilderID
func fetchBlueprintContent(s *discordgo.Session, thread *discordgo.Channel) threadData {
	msgs, err := s.ChannelMessages(thread.ID, 1, "", "0", "")
	if err != nil || len(msgs) == 0 {
		slog.Warn("fetching blueprint messages", "thread", thread.ID, "err", err)
		return threadData{}
	}

	parsed := blueprint.ParseFirstPost(msgs[0].Content)
	authorID := msgs[0].Author.ID

	allowedIDs := map[string]bool{authorID: true}
	sections, screenshots, videos, lastContributorTime := collectMedia(s, thread.ID, allowedIDs)

	firstPostMsg := msgs[0].Timestamp

	return threadData{
		ID:                  parsed.BuilderID,
		GuildName:           parsed.Description,  // repurposed
		AuthorID:            authorID,
		Screenshots:         screenshots,
		ScreenshotSections:  sections,
		Videos:              videos,
		Lore:                parsed.Materials,     // repurposed
		WhatToVisit:         parsed.Price,         // repurposed
		BuildTitle:          parsed.BuilderName,   // repurposed
		FirstPostTime:       firstPostMsg,
		LastContributorTime: lastContributorTime,
	}
}

func blueprintHasChanged(prev, next blueprint.Blueprint) bool {
	return prev.Price != next.Price ||
		prev.Materials != next.Materials ||
		prev.Description != next.Description ||
		prev.IsFree != next.IsFree ||
		prev.IsPayToBuild != next.IsPayToBuild ||
		prev.BuilderName != next.BuilderName ||
		len(prev.Screenshots) != len(next.Screenshots) ||
		len(prev.Videos) != len(next.Videos)
}
