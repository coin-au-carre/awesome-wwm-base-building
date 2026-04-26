package discord

import "ruby/internal/guild"

const (
	abuseMinVotedThreads   = 4 // minimum distinct guilds voted on before flagging (matches weight ×1 threshold)
	abuseHighScoreThreshold = 4 // minimum raw pts to a guild to count as genuine engagement (⭐ + at least 2 others)
	abuseMinHighScoreOthers = 1 // voter must give ≥ abuseHighScoreThreshold pts to at least this many OTHER guilds
)

// AbuseFlag describes a voter who gave max reactions to one guild but only low reactions elsewhere.
type AbuseFlag struct {
	UserID          string
	Threads         int    // distinct threads voted on
	TopThreadID     string // thread receiving the most raw points
	TopGuildName    string // resolved name for TopThreadID
	TopRawPts       int    // raw pts given to top guild
	TotalRawPts     int    // raw pts across all guilds
	HighScoreOthers int    // other guilds (excl. top) where voter gave ≥ abuseHighScoreThreshold pts
	Cap             int    // ceil(avg pts on other guilds) — score is capped to this on their top guild
}

// rawPointsPerUser returns a map of userID → raw (unweighted) reaction points for one thread.
// Thumbs-up skin-tone variants are deduplicated so a voter scores at most 1 pt from that emoji.
func rawPointsPerUser(reactions map[string][]string) map[string]int {
	out := make(map[string]int)
	thumbsVoters := make(map[string]bool)
	for emoji, users := range reactions {
		switch emoji {
		case "⭐":
			for _, uid := range users {
				out[uid] += scorePerStar
			}
		case "👍", "👍🏻", "👍🏼", "👍🏽", "👍🏾", "👍🏿":
			for _, uid := range users {
				thumbsVoters[uid] = true
			}
		case "🔥", "❤️":
			for _, uid := range users {
				out[uid] += scorePerLike
			}
		}
	}
	for uid := range thumbsVoters {
		out[uid] += scorePerLike
	}
	return out
}

// detectVoterAbuse scans all fetched threads and flags voters with abnormal boost patterns.
// Used during sync when raw fetchedThread data is available.
func detectVoterAbuse(threads []fetchedThread, guildNameByThreadID map[string]string) []AbuseFlag {
	rm := make(guild.ReactionMap, len(threads))
	for _, ft := range threads {
		rm[ft.thread.ID] = ft.reactions
	}
	return DetectVoterAbuseFromReactionsWithThresholds(rm, guildNameByThreadID, abuseMinVotedThreads, abuseMinHighScoreOthers)
}

// VoterStats holds per-voter reaction totals across all threads.
type VoterStats struct {
	UserID          string
	Threads         int     // distinct threads voted on
	TotalRawPts     int     // raw points across all guilds
	TopThreadID     string  // thread receiving the most raw points
	TopGuildName    string  // resolved name for TopThreadID
	TopRawPts       int     // raw points given to the top guild
	HighScoreOthers int     // other guilds (excl. top) where voter gave ≥ abuseHighScoreThreshold pts
	Cap             int     // ceil(avg pts on other guilds)
}

// ceilDiv returns ceil(a/b) for positive integers.
func ceilDiv(a, b int) int {
	return (a + b - 1) / b
}

// SummarizeVoters returns per-voter stats for every voter found in the ReactionMap.
func SummarizeVoters(reactions guild.ReactionMap, guildNameByThreadID map[string]string) []VoterStats {
	userPoints := make(map[string]map[string]int)
	for threadID, emojiMap := range reactions {
		for uid, p := range rawPointsPerUser(emojiMap) {
			if p == 0 {
				continue
			}
			if userPoints[uid] == nil {
				userPoints[uid] = make(map[string]int)
			}
			userPoints[uid][threadID] += p
		}
	}

	out := make([]VoterStats, 0, len(userPoints))
	for uid, byThread := range userPoints {
		total, topPts := 0, 0
		topID := ""
		for tid, pts := range byThread {
			total += pts
			if pts > topPts {
				topPts = pts
				topID = tid
			}
		}
		highOthers := 0
		for tid, pts := range byThread {
			if tid != topID && pts >= abuseHighScoreThreshold {
				highOthers++
			}
		}
		cap := 0
		if len(byThread) > 1 {
			cap = ceilDiv(total-topPts, len(byThread)-1)
		}
		out = append(out, VoterStats{
			UserID:          uid,
			Threads:         len(byThread),
			TotalRawPts:     total,
			TopThreadID:     topID,
			TopGuildName:    guildNameByThreadID[topID],
			TopRawPts:       topPts,
			HighScoreOthers: highOthers,
			Cap:             cap,
		})
	}
	return out
}

// DetectVoterAbuseFromReactions runs abuse detection against a saved ReactionMap
// using the default thresholds.
func DetectVoterAbuseFromReactions(reactions guild.ReactionMap, guildNameByThreadID map[string]string) []AbuseFlag {
	return DetectVoterAbuseFromReactionsWithThresholds(reactions, guildNameByThreadID, abuseMinVotedThreads, abuseMinHighScoreOthers)
}

// DetectVoterAbuseFromReactionsWithThresholds flags voters who:
//   - voted on at least minVotedThreads guilds (has voting weight)
//   - gave ≥ abuseHighScoreThreshold pts to their top guild
//   - gave ≥ abuseHighScoreThreshold pts to fewer than minHighScoreOthers other guilds
func DetectVoterAbuseFromReactionsWithThresholds(reactions guild.ReactionMap, guildNameByThreadID map[string]string, minVotedThreads, minHighScoreOthers int) []AbuseFlag {
	all := SummarizeVoters(reactions, guildNameByThreadID)
	var flags []AbuseFlag
	for _, s := range all {
		if s.Threads >= minVotedThreads &&
			s.TopRawPts >= abuseHighScoreThreshold &&
			s.HighScoreOthers < minHighScoreOthers {
			flags = append(flags, AbuseFlag{
				UserID:          s.UserID,
				Threads:         s.Threads,
				TopThreadID:     s.TopThreadID,
				TopGuildName:    s.TopGuildName,
				TopRawPts:       s.TopRawPts,
				TotalRawPts:     s.TotalRawPts,
				HighScoreOthers: s.HighScoreOthers,
				Cap:             s.Cap,
			})
		}
	}
	return flags
}
