package discord

import (
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

const numWorkers = 20

type SyncStats struct {
	Total                int
	New                  int
	Updated              int
	NewNames             []string
	UpdatedNames         []string
	MoreScreenshotNames  []string          // existing entries that gained screenshots
	NewThreadLinks       map[string]string // guild name → discord thread URL
	VoterGuildCounts     map[string]int    // userID → number of distinct guilds voted on
	DuplicateWarnings    []string
}

type SyncConfig struct {
	ForumChannelID string
	DryRun         bool
}

type threadData struct {
	ID                 string
	GuildName          string
	AuthorID           string
	Builders           []string
	Score              int
	Screenshots        []string
	ScreenshotSections []guild.ScreenshotSection
	Videos             []string
	Lore               string
	WhatToVisit        string
	CoverIdx           int // 1-based; 0 = not set
}

type fetchedThread struct {
	idx       int
	thread    *discordgo.Channel
	data      threadData
	reactions map[string][]string
}

// SyncFetchResult holds all raw data from SyncFetch, ready for SyncFinalize.
type SyncFetchResult struct {
	Guilds      []guild.Guild
	VoterCounts map[string]int // userID → distinct thread count (this channel only)
	threads     []fetchedThread
	newIndices  map[int]bool
	tagMap      map[string]string
	Stats       SyncStats // partial: New, NewNames, NewThreadLinks filled; Updated/Total set by Finalize
}

// SyncFetch collects threads, fetches content and reactions, and returns raw results.
// Call SyncFinalize with merged cross-channel voter weights to complete scoring.
func SyncFetch(b *Bot, guilds []guild.Guild, cfg SyncConfig) (SyncFetchResult, error) {
	forumChannel, err := b.Session.Channel(cfg.ForumChannelID)
	if err != nil {
		return SyncFetchResult{}, fmt.Errorf("fetching channel: %w", err)
	}

	tagMap := make(map[string]string, len(forumChannel.AvailableTags))
	for _, tag := range forumChannel.AvailableTags {
		tagMap[tag.ID] = tag.Name
	}
	slog.Info("forum tags loaded", "count", len(tagMap))

	t0 := time.Now()
	threads, err := collectThreads(b.Session, cfg.ForumChannelID, forumChannel.GuildID)
	if err != nil {
		return SyncFetchResult{}, err
	}
	slog.Info("threads collected", "count", len(threads), "elapsed", time.Since(t0).Round(time.Millisecond))

	var partialStats SyncStats
	partialStats.NewThreadLinks = make(map[string]string)
	guildMap := buildGuildMap(guilds)
	newIndices := make(map[int]bool)

	for _, thread := range threads {
		name, threadID := guild.ExtractNameAndID(thread.Name)
		key := strings.ToLower(name)
		if _, exists := guildMap[key]; !exists {
			newThreadLink := fmt.Sprintf("https://discord.com/channels/%s/%s", thread.GuildID, thread.ID)

			// Check for similar existing guild names before adding.
			for _, existing := range guilds {
				existingName := guild.ExtractName(existing.Name)
				if similarGuildName(name, existingName) {
					warning := fmt.Sprintf(
						"⚠️ **Possible duplicate guild detected:**\n• New: **%s** → %s\n• Existing: **%s** → %s",
						name, newThreadLink, existingName, existing.DiscordThread,
					)
					slog.Warn("possible duplicate guild", "new", name, "existing", existingName)
					partialStats.DuplicateWarnings = append(partialStats.DuplicateWarnings, warning)
					if !cfg.DryRun {
						b.Notify(warning)
					}
				}
			}

			idx := len(guilds)
			guilds = append(guilds, guild.Guild{Name: name, ID: threadID, Builders: []string{}})
			guildMap[key] = len(guilds) - 1
			newIndices[idx] = true
			partialStats.New++
			partialStats.NewNames = append(partialStats.NewNames, name)
			partialStats.NewThreadLinks[name] = newThreadLink
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
		for _, users := range r.reactions {
			for _, uid := range users {
				if userThreads[uid] == nil {
					userThreads[uid] = make(map[string]bool)
				}
				userThreads[uid][r.thread.ID] = true
			}
		}
	}
	slog.Info("content and reactions fetched", "threads", len(fetched), "elapsed", time.Since(tFetch).Round(time.Millisecond))

	voterCounts := make(map[string]int, len(userThreads))
	for uid, threadSet := range userThreads {
		voterCounts[uid] = len(threadSet)
	}

	return SyncFetchResult{
		Guilds:      guilds,
		VoterCounts: voterCounts,
		threads:     fetched,
		newIndices:  newIndices,
		tagMap:      tagMap,
		Stats:       partialStats,
	}, nil
}

// SyncFinalize scores all fetched threads using the provided merged voter weights
// and returns the final guild list and complete stats.
func SyncFinalize(result SyncFetchResult, voterWeights map[string]int) ([]guild.Guild, SyncStats) {
	slog.Info("voter weights applied", "voters", len(voterWeights))

	guilds := result.Guilds
	stats := result.Stats
	now := time.Now().UTC().Format("January 2, 2006 at 03:04 PM UTC")

	for _, r := range result.threads {
		data := r.data
		data.Score = computeScore(r.reactions, voterWeights, data.Lore, data.WhatToVisit)

		var tags []string
		for _, tagID := range r.thread.AppliedTags {
			if name, ok := result.tagMap[tagID]; ok {
				tags = append(tags, name)
			}
		}

		g := guilds[r.idx]
		// Normalize stored name if it contains an embedded ID (e.g. "WITCHERS [10248427").
		if cleanName, embeddedID := guild.ExtractNameAndID(g.Name); cleanName != g.Name {
			g.Name = cleanName
			if g.ID == "" && embeddedID != "" {
				g.ID = embeddedID
			}
		}
		if g.ID == "" && data.ID != "" {
			g.ID = data.ID
		}
		if g.ID == "" {
			if _, threadID := guild.ExtractNameAndID(r.thread.Name); threadID != "" {
				g.ID = threadID
			}
		}
		if data.GuildName != "" {
			if strings.EqualFold(data.GuildName, g.Name) {
				g.GuildName = ""
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
		g.ScreenshotSections = data.ScreenshotSections
		g.Videos = data.Videos
		if idx := data.CoverIdx; idx >= 1 && idx <= len(data.Screenshots) {
			g.CoverImage = data.Screenshots[idx-1]
		} else {
			g.CoverImage = ""
		}
		g.Lore = data.Lore
		g.WhatToVisit = data.WhatToVisit
		if data.AuthorID != "" {
			g.BuilderDiscordID = data.AuthorID
		}

		isNew := result.newIndices[r.idx]
		if !isNew && hasChanged(prev, g) {
			stats.Updated++
			stats.UpdatedNames = append(stats.UpdatedNames, g.Name)
			g.LastModified = now
			if len(g.Screenshots) > len(prev.Screenshots) {
				stats.MoreScreenshotNames = append(stats.MoreScreenshotNames, g.Name)
			}
		} else if isNew {
			g.LastModified = now
		}

		guilds[r.idx] = g
		slog.Info("guild scored", "name", g.Name, "score", g.Score, "tags", strings.Join(g.Tags, ", "))
	}

	stats.Total = len(guilds)
	stats.VoterGuildCounts = result.VoterCounts
	return guilds, stats
}

func collectThreads(s *discordgo.Session, forumChannelID, guildID string) ([]*discordgo.Channel, error) {
	var (
		active    *discordgo.ThreadsList
		archived  *discordgo.ThreadsList
		activeErr error
		wg        sync.WaitGroup
	)
	wg.Add(2)
	go func() { defer wg.Done(); active, activeErr = s.GuildThreadsActive(guildID) }()
	go func() {
		defer wg.Done()
		var err error
		archived, err = s.ThreadsArchived(forumChannelID, nil, 0)
		if err != nil {
			slog.Warn("fetching archived threads", "err", err)
		}
	}()
	wg.Wait()

	if activeErr != nil {
		return nil, fmt.Errorf("fetching active threads: %w", activeErr)
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

	id, guildName, builders, lore, whatToVisit, coverIdx := guild.ParseFirstPost(msgs[0].Content)
	authorID := msgs[0].Author.ID
	sections, screenshots, videos := collectMedia(s, thread.ID, authorID)

	return threadData{
		ID:                 id,
		GuildName:          guildName,
		AuthorID:           authorID,
		Builders:           resolveBuilders(s, builders),
		Screenshots:        screenshots,
		ScreenshotSections: sections,
		Videos:             videos,
		Lore:               lore,
		WhatToVisit:        whatToVisit,
		CoverIdx:           coverIdx,
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

func isSupportedVideoURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.Hostname() == "www.tiktok.com" || u.Hostname() == "tiktok.com"
}

var reSectionHeader = regexp.MustCompile(`^(#{1,3})\s+(.+)`)

func collectMedia(s *discordgo.Session, threadID, authorID string) (sections []guild.ScreenshotSection, screenshots, videos []string) {
	seen := make(map[string]bool)
	var lastID string
	var currentSection *guild.ScreenshotSection

	addImage := func(url string) {
		if currentSection == nil {
			sections = append(sections, guild.ScreenshotSection{})
			currentSection = &sections[len(sections)-1]
		}
		currentSection.Screenshots = append(currentSection.Screenshots, url)
		screenshots = append(screenshots, url)
	}

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
			if m := reSectionHeader.FindStringSubmatch(strings.TrimSpace(msg.Content)); len(m) == 3 {
				label := strings.TrimSpace(m[2])
				if currentSection != nil && currentSection.Label == "" && len(currentSection.Screenshots) > 0 {
					// Builder posted images first then captioned them — label the preceding batch.
					currentSection.Label = label
				} else {
					sections = append(sections, guild.ScreenshotSection{Label: label})
					currentSection = &sections[len(sections)-1]
				}
			}
			for _, att := range msg.Attachments {
				if seen[att.URL] {
					continue
				}
				seen[att.URL] = true
				if guild.IsImage(att.Filename) {
					addImage(att.URL)
					slog.Debug("screenshot found", "thread", threadID, "url", att.URL)
				} else if guild.IsVideo(att.Filename) {
					videos = append(videos, att.URL)
					slog.Debug("video found", "thread", threadID, "url", att.URL)
				}
			}
			for _, embed := range msg.Embeds {
				isVideo := embed.Type == discordgo.EmbedTypeVideo || isSupportedVideoURL(embed.URL)
				if isVideo && embed.URL != "" && !seen[embed.URL] {
					seen[embed.URL] = true
					videos = append(videos, embed.URL)
					slog.Debug("embed video found", "thread", threadID, "url", embed.URL)
				} else if embed.Image != nil && embed.Image.URL != "" && !seen[embed.Image.URL] {
					seen[embed.Image.URL] = true
					addImage(embed.Image.URL)
					slog.Debug("embed image found", "thread", threadID, "url", embed.Image.URL)
				}
			}
		}
		lastID = msgs[len(msgs)-1].ID
		if len(msgs) < 100 {
			break
		}
	}
	for i := range sections {
		slices.Reverse(sections[i].Screenshots)
	}
	slices.Reverse(sections)
	slices.Reverse(screenshots)
	slices.Reverse(videos)
	// Drop sections that ended up with no screenshots (e.g. a section header
	// in the first post that was processed after all images were already
	// assigned to a labelled section).
	filtered := sections[:0]
	for _, sec := range sections {
		if len(sec.Screenshots) > 0 {
			filtered = append(filtered, sec)
		}
	}
	sections = filtered
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
		m[strings.ToLower(guild.ExtractName(g.Name))] = i
	}
	return m
}

// similarGuildName returns true if two guild names are likely the same guild.
// It normalizes both names (lowercase, alphanumeric only) then checks for
// substring containment or a Levenshtein distance ≤ 2.
func similarGuildName(a, b string) bool {
	na, nb := normalizeName(a), normalizeName(b)
	if na == nb {
		return false // identical keys are already deduplicated by the guildMap
	}
	if len(na) == 0 || len(nb) == 0 {
		return false
	}
	// Containment: "thewitchers" contains "witchers"
	if strings.Contains(na, nb) || strings.Contains(nb, na) {
		return true
	}
	// Edit distance for names that aren't too short (avoid false positives).
	// Allow at most 1 edit per 5 characters (floor), so short names require
	// near-exact matches while longer names tolerate more variation.
	minLen := len(na)
	if len(nb) < minLen {
		minLen = len(nb)
	}
	if minLen >= 5 && levenshtein(na, nb) <= minLen/5 {
		return true
	}
	return false
}

func normalizeName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	row := make([]int, lb+1)
	for j := range row {
		row[j] = j
	}
	for i := 1; i <= la; i++ {
		prev := i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			next := min(min(row[j]+1, prev+1), row[j-1]+cost)
			row[j-1] = prev
			prev = next
		}
		row[lb] = prev
	}
	return row[lb]
}

func hasChanged(prev, next guild.Guild) bool {
	return prev.Score != next.Score ||
		prev.Lore != next.Lore ||
		prev.WhatToVisit != next.WhatToVisit ||
		prev.GuildName != next.GuildName ||
		prev.CoverImage != next.CoverImage ||
		len(prev.Screenshots) != len(next.Screenshots) ||
		len(prev.Videos) != len(next.Videos) ||
		!slices.Equal(prev.Builders, next.Builders) ||
		!slices.Equal(prev.Tags, next.Tags)
}
