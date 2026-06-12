package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"ruby/internal/guild"
	"ruby/internal/interior"

	"github.com/bwmarrin/discordgo"
)

// InteriorSyncFetchResult holds raw data from an interior design forum channel fetch.
type InteriorSyncFetchResult struct {
	Interiors   []interior.Interior
	VoterCounts map[string]int
	Users       guild.UserMap
	threads     []fetchedThread
	newIndices  map[int]bool
	Stats       SyncStats
}

// InteriorSyncFetch collects all threads from an interior design forum channel.
func InteriorSyncFetch(b *Bot, interiors []interior.Interior, cfg SyncConfig) (InteriorSyncFetchResult, error) {
	forumChannel, err := b.Session.Channel(cfg.ForumChannelID)
	if err != nil {
		return InteriorSyncFetchResult{}, fmt.Errorf("fetching channel: %w", err)
	}
	slog.Info("interior forum channel loaded", "name", forumChannel.Name)

	t0 := time.Now()
	threads, err := collectThreads(b.Session, cfg.ForumChannelID, forumChannel.GuildID)
	if err != nil {
		return InteriorSyncFetchResult{}, err
	}
	slog.Info("interior threads collected", "count", len(threads), "elapsed", time.Since(t0).Round(time.Millisecond))

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

	// Deduplicate existing interiors by discordThread.
	seen := make(map[string]int, len(interiors))
	deduped := interiors[:0]
	for _, it := range interiors {
		if it.DiscordThread == "" {
			deduped = append(deduped, it)
			continue
		}
		if prev, dup := seen[it.DiscordThread]; dup {
			slog.Warn("duplicate interior discordThread removed", "url", it.DiscordThread, "kept", interiors[prev].Name, "removed", it.Name)
			deduped[prev] = it
			continue
		}
		seen[it.DiscordThread] = len(deduped)
		deduped = append(deduped, it)
	}
	interiors = deduped

	threadURLToIdx := make(map[string]int, len(interiors))
	for i, it := range interiors {
		if it.DiscordThread != "" {
			threadURLToIdx[it.DiscordThread] = i
		}
	}
	newIndices := make(map[int]bool)

	for _, thread := range threads {
		threadURL := fmt.Sprintf("https://discord.com/channels/%s/%s", thread.GuildID, thread.ID)
		if urlIdx, exists := threadURLToIdx[threadURL]; exists {
			storedName := interiors[urlIdx].Name
			newName := guild.ExtractName(thread.Name)
			if !strings.EqualFold(storedName, newName) && newName != "" {
				slog.Info("interior renamed", "old", storedName, "new", newName)
				interiors[urlIdx].Name = newName
			}
			continue
		}
		name := guild.ExtractName(thread.Name)
		idx := len(interiors)
		interiors = append(interiors, interior.Interior{Name: name, DiscordThread: threadURL})
		threadURLToIdx[threadURL] = idx
		newIndices[idx] = true
		partialStats.New++
		partialStats.NewNames = append(partialStats.NewNames, name)
		partialStats.NewThreadLinks[name] = threadURL
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
					data = fetchInteriorContent(b.Session, j.thread)
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
	slog.Info("interior content and reactions fetched", "threads", len(fetched), "elapsed", time.Since(tFetch).Round(time.Millisecond))

	voterCounts := make(map[string]int, len(userThreads))
	for uid, threadSet := range userThreads {
		voterCounts[uid] = len(threadSet)
	}

	userCache := resolveUsers(b.Session, guildID(fetched), userThreads)

	return InteriorSyncFetchResult{
		Interiors:   interiors,
		VoterCounts: voterCounts,
		Users:       userCache,
		threads:     fetched,
		newIndices:  newIndices,
		Stats:       partialStats,
	}, nil
}

// InteriorSyncFinalize updates interiors from fetched content and returns the final list plus stats.
func InteriorSyncFinalize(result InteriorSyncFetchResult) ([]interior.Interior, SyncStats) {
	interiors := result.Interiors
	stats := result.Stats
	now := guild.ModifiedNow()

	for _, r := range result.threads {
		it := interiors[r.idx]
		prev := it

		name := guild.ExtractName(r.thread.Name)
		if name != "" {
			it.Name = name
		}
		it.Description = r.data.GuildName
		it.Screenshots = r.data.Screenshots
		it.Videos = r.data.Videos
		if it.CreatedAt == "" {
			it.CreatedAt = r.data.FirstPostTime.UTC().Format(guild.ModifiedLayout)
		}
		if strings.Contains(strings.ToLower(it.Name), "all builders") {
			it.BuilderName = "anyone"
			it.BuilderID = ""
		} else if r.data.AuthorID != "" {
			it.BuilderID = r.data.AuthorID
			if u, ok := result.Users[r.data.AuthorID]; ok {
				it.BuilderName = u.DisplayName()
			}
		}

		isNew := result.newIndices[r.idx]
		if !isNew && interiorHasChanged(prev, it) {
			stats.Updated++
			stats.UpdatedNames = append(stats.UpdatedNames, it.Name)
			if len(it.Screenshots) > len(prev.Screenshots) && !screenshotOnCooldown(prev.LastScreenshotNotifiedAt) {
				stats.MoreScreenshotNames = append(stats.MoreScreenshotNames, it.Name)
				it.LastScreenshotNotifiedAt = now
				it.LastModified = now
			}
			if len(it.Videos) > len(prev.Videos) && !screenshotOnCooldown(prev.LastVideoNotifiedAt) {
				stats.MoreVideoNames = append(stats.MoreVideoNames, it.Name)
				it.LastVideoNotifiedAt = now
				it.LastModified = now
			}
		} else if isNew {
			it.LastModified = now
		}

		interiors[r.idx] = it
		slog.Info("interior updated", "name", it.Name)
	}

	stats.Total = len(interiors)
	return interiors, stats
}

// fetchInteriorContent fetches the first post author and all media from an interior thread.
func fetchInteriorContent(s *discordgo.Session, thread *discordgo.Channel) threadData {
	msgs, err := s.ChannelMessages(thread.ID, 1, "", "0", "")
	if err != nil || len(msgs) == 0 {
		slog.Warn("fetching interior messages", "thread", thread.ID, "err", err)
		return threadData{}
	}

	authorID := msgs[0].Author.ID
	firstPostTime := msgs[0].Timestamp
	firstPostText := msgs[0].Content

	// For "all builders" threads collect from everyone; otherwise only the author.
	var allowedIDs map[string]bool
	if !strings.Contains(strings.ToLower(thread.Name), "all builders") {
		allowedIDs = map[string]bool{authorID: true}
	}
	_, screenshots, videos, lastContributorTime := collectMedia(s, thread.ID, allowedIDs)

	return threadData{
		AuthorID:            authorID,
		GuildName:           firstPostText, // reused for description
		Screenshots:         screenshots,
		Videos:              videos,
		FirstPostTime:       firstPostTime,
		LastContributorTime: lastContributorTime,
	}
}

func interiorHasChanged(prev, next interior.Interior) bool {
	return prev.BuilderName != next.BuilderName ||
		prev.Description != next.Description ||
		len(prev.Screenshots) != len(next.Screenshots) ||
		len(prev.Videos) != len(next.Videos)
}
