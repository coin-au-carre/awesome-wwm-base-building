package discord

import (
	"fmt"
	"log/slog"
	"strings"
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

	// Deduplicate existing blueprints by discordThread — keep the last occurrence of each URL.
	seen := make(map[string]int, len(blueprints))
	deduped := blueprints[:0]
	for _, bp := range blueprints {
		if bp.DiscordThread == "" {
			deduped = append(deduped, bp)
			continue
		}
		if prev, dup := seen[bp.DiscordThread]; dup {
			slog.Warn("duplicate blueprint discordThread removed", "url", bp.DiscordThread, "kept", blueprints[prev].Name, "removed", bp.Name)
			deduped[prev] = bp // replace with later entry (more up-to-date)
			continue
		}
		seen[bp.DiscordThread] = len(deduped)
		deduped = append(deduped, bp)
	}
	blueprints = deduped

	// Build URL → index map for existing blueprints.
	threadURLToIdx := make(map[string]int, len(blueprints))
	for i, bp := range blueprints {
		if bp.DiscordThread != "" {
			threadURLToIdx[bp.DiscordThread] = i
		}
	}
	newIndices := make(map[int]bool)

	seenThreadURLs := make(map[string]bool, len(threads))
	for _, thread := range threads {
		threadURL := fmt.Sprintf("https://discord.com/channels/%s/%s", thread.GuildID, thread.ID)
		seenThreadURLs[threadURL] = true
		if urlIdx, exists := threadURLToIdx[threadURL]; exists {
			// Check for rename: same thread URL, different name.
			storedName := blueprints[urlIdx].Name
			newName := guild.ExtractName(thread.Name)
			if !strings.EqualFold(storedName, newName) && newName != "" {
				slog.Info("blueprint renamed", "old", storedName, "new", newName)
				notice := fmt.Sprintf("ℹ️ **Blueprint renamed:** **%s** → **%s**\n%s", storedName, newName, threadURL)
				partialStats.DuplicateWarnings = append(partialStats.DuplicateWarnings, notice)
				blueprints[urlIdx].Name = newName
			}
			continue
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

	// Tombstone entries whose thread no longer appears in the forum (deleted/closed).
	if cfg.GuildFilter == "" {
		for i := range blueprints {
			url := blueprints[i].DiscordThread
			if url != "" && !seenThreadURLs[url] && !blueprints[i].Deleted {
				slog.Info("blueprint thread gone, marking deleted", "name", blueprints[i].Name, "url", url)
				blueprints[i].Deleted = true
			}
		}
	}

	fetched, userThreads := fetchAllContent(b, threads, threadURLToIdx, fetchBlueprintContent, "blueprint")

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
		if blueprints[r.idx].Deleted {
			continue
		}
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
		// Only overwrite media on a successful fetch; a mid-pagination API failure returns an
		// empty result that would otherwise look like every screenshot got deleted.
		if !r.data.MediaOK {
			slog.Warn("skipping media update for blueprint due to failed media fetch", "name", bp.Name)
		} else {
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
		hasFreeTag := containsTag(tags, "Free")
		hasPaidTag := containsTag(tags, "Paid")
		if hasFreeTag || hasPaidTag {
			// Forum "Free"/"Paid" tags take dominance over the legacy title/price-text parsing below.
			bp.IsFree = hasFreeTag
			bp.IsPayToBuild = hasPaidTag
		} else if r.data.WhatToVisit != "" {
			bp.Price = r.data.WhatToVisit
			lower := strings.ToLower(bp.Price)
			bp.IsFree = strings.Contains(lower, "free")
			bp.IsPayToBuild = lower != "free" && lower != ""
		} else {
			descLower := strings.ToLower(bp.Description)
			if blueprint.RePayInProse.MatchString(bp.Description) {
				bp.IsPayToBuild = true
			} else if strings.Contains(descLower, "free") {
				bp.IsFree = true
			} else {
				bp.IsPayToBuild = true // no "free" anywhere → paid by default
			}
			// Thread title keywords override first-post price parsing (legacy system only).
			titleLower := strings.ToLower(r.thread.Name)
			if strings.Contains(titleLower, "free") {
				bp.IsFree = true
			}
			if strings.Contains(titleLower, "paid") {
				bp.IsPayToBuild = true
			}
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
		if len(r.data.ShareCodes) > 0 {
			bp.ShareCodes = r.data.ShareCodes
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
	sections, screenshots, videos, lastContributorTime, mediaOK := collectMedia(s, thread.ID, allowedIDs)

	firstPostMsg := msgs[0].Timestamp

	return threadData{
		ID:                  parsed.BuilderID,
		GuildName:           parsed.Description, // repurposed
		AuthorID:            authorID,
		Screenshots:         screenshots,
		ScreenshotSections:  sections,
		Videos:              videos,
		Lore:                parsed.Materials,   // repurposed
		WhatToVisit:         parsed.Price,       // repurposed
		BuildTitle:          parsed.BuilderName, // repurposed
		ShareCodes:          parsed.ShareCodes,
		CoverIdx:            parsed.CoverIdx,
		FirstPostTime:       firstPostMsg,
		LastContributorTime: lastContributorTime,
		MediaOK:             mediaOK,
	}
}

func containsTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

func blueprintHasChanged(prev, next blueprint.Blueprint) bool {
	return prev.Price != next.Price ||
		prev.Materials != next.Materials ||
		prev.Description != next.Description ||
		prev.IsFree != next.IsFree ||
		prev.IsPayToBuild != next.IsPayToBuild ||
		prev.BuilderName != next.BuilderName ||
		len(prev.ShareCodes) != len(next.ShareCodes) ||
		len(prev.Screenshots) != len(next.Screenshots) ||
		len(prev.Videos) != len(next.Videos)
}
