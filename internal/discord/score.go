package discord

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

const (
	scorePerStar    = 2
	scorePerLike    = 1
	scorePerFire    = 1
	scoreLoreBonus  = 1
	scoreVisitBonus = 1
)

var scoredEmojis = []string{
	"⭐",
	"👍", "👍🏻", "👍🏼", "👍🏽", "👍🏾", "👍🏿",
	"🔥",
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
// threads they reacted to across all channels.
func voterWeight(distinctThreads int) int {
	switch {
	case distinctThreads >= 12:
		return 3
	case distinctThreads >= 6:
		return 2
	case distinctThreads >= 2:
		return 1
	default:
		return 0
	}
}

// MergeVoterCounts combines two voter count maps by summing per-user thread counts.
// Used to share voter weight across guild and solo channels.
func MergeVoterCounts(a, b map[string]int) map[string]int {
	merged := make(map[string]int, len(a)+len(b))
	for uid, count := range a {
		merged[uid] += count
	}
	for uid, count := range b {
		merged[uid] += count
	}
	return merged
}

// ComputeVoterWeights converts per-user thread counts into weight multipliers.
func ComputeVoterWeights(counts map[string]int) map[string]int {
	weights := make(map[string]int, len(counts))
	for uid, count := range counts {
		if w := voterWeight(count); w > 0 {
			weights[uid] = w
		}
	}
	return weights
}

// CollectVoterCounts fetches only reactions (no content/media) from all threads
// in a forum channel and returns a map of userID → distinct thread count.
func CollectVoterCounts(b *Bot, forumChannelID string) (map[string]int, error) {
	ch, err := b.Session.Channel(forumChannelID)
	if err != nil {
		return nil, err
	}
	threads, err := collectThreads(b.Session, forumChannelID, ch.GuildID)
	if err != nil {
		return nil, err
	}

	type result struct {
		threadID  string
		reactions map[string][]string
	}

	jobs := make(chan string, len(threads))
	results := make(chan result, len(threads))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for tid := range jobs {
				results <- result{tid, fetchThreadReactions(b.Session, tid)}
			}
		}()
	}
	for _, t := range threads {
		jobs <- t.ID
	}
	close(jobs)
	go func() { wg.Wait(); close(results) }()

	userThreads := make(map[string]map[string]bool)
	for r := range results {
		for _, users := range r.reactions {
			for _, uid := range users {
				if userThreads[uid] == nil {
					userThreads[uid] = make(map[string]bool)
				}
				userThreads[uid][r.threadID] = true
			}
		}
	}

	counts := make(map[string]int, len(userThreads))
	for uid, threads := range userThreads {
		counts[uid] = len(threads)
	}
	return counts, nil
}

// fetchThreadReactions fetches all reactor user IDs for each scored emoji in a single thread.
// All emojis are fetched in parallel. Returns emoji → []userID.
func fetchThreadReactions(s *discordgo.Session, threadID string) map[string][]string {
	type result struct {
		emoji string
		ids   []string
	}

	ch := make(chan result, len(scoredEmojis))
	var wg sync.WaitGroup
	for _, emoji := range scoredEmojis {
		wg.Add(1)
		go func(emoji string) {
			defer wg.Done()
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
				ch <- result{emoji, ids}
			}
		}(emoji)
	}
	wg.Wait()
	close(ch)

	reactions := make(map[string][]string)
	for r := range ch {
		reactions[r.emoji] = r.ids
	}
	return reactions
}
