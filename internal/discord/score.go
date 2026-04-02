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
// guilds they reacted to
func voterWeight(distinctGuilds int) int {
	switch {
	case distinctGuilds >= 12:
		return 3
	case distinctGuilds >= 6:
		return 2
	case distinctGuilds >= 2:
		return 1
	default:
		return 0
	}
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
