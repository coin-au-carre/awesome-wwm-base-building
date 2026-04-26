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
	"❤️",
}

func filterReactions(reactions map[string][]string, blacklist map[string]bool) map[string][]string {
	if len(blacklist) == 0 {
		return reactions
	}
	out := make(map[string][]string, len(reactions))
	for emoji, users := range reactions {
		var filtered []string
		for _, uid := range users {
			if !blacklist[uid] {
				filtered = append(filtered, uid)
			}
		}
		if len(filtered) > 0 {
			out[emoji] = filtered
		}
	}
	return out
}

// computeScore tallies weighted reaction points for a thread.
// rawCaps optionally limits how many raw points a specific voter can contribute (before weight).
// Pass nil for no caps.
func computeScore(reactions map[string][]string, weights map[string]int, rawCaps map[string]int, lore, whatToVisit string) int {
	// Accumulate raw points per voter, then apply weight and optional cap.
	userRaw := make(map[string]int)
	thumbsVoters := make(map[string]bool)
	for emoji, users := range reactions {
		switch emoji {
		case "⭐":
			for _, uid := range users {
				userRaw[uid] += scorePerStar
			}
		case "👍", "👍🏻", "👍🏼", "👍🏽", "👍🏾", "👍🏿":
			for _, uid := range users {
				thumbsVoters[uid] = true
			}
		case "🔥", "❤️":
			for _, uid := range users {
				userRaw[uid] += scorePerLike
			}
		}
	}
	for uid := range thumbsVoters {
		userRaw[uid] += scorePerLike
	}

	score := 0
	for uid, raw := range userRaw {
		if cap, ok := rawCaps[uid]; ok && raw > cap {
			raw = cap
		}
		score += raw * weights[uid]
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
	case distinctThreads >= 8:
		return 2
	case distinctThreads >= 4:
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
