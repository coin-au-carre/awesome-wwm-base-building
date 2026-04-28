package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

// VoteRecord represents a single user's voting activity on one guild thread.
type VoteRecord struct {
	GuildName    string
	ThreadID     string
	Username     string
	UserID       string
	Emojis       []string // all emojis this user reacted with on this thread
	ThreadsVoted int      // distinct guild threads this user reacted to
	Weight       float64  // computed vote weight
	Points       float64  // weighted points contributed to this guild
}

// AnalyzeVoters fetches all reactions from the guild forum channel and returns
// a flat list of per-voter, per-guild records. Voter weights are computed from
// guild-only thread counts (same formula as the real sync but guild-scoped).
func AnalyzeVoters(b *Bot, forumChannelID string) ([]VoteRecord, error) {
	forumChannel, err := b.Session.Channel(forumChannelID)
	if err != nil {
		return nil, fmt.Errorf("fetching channel: %w", err)
	}

	threads, err := collectThreads(b.Session, forumChannelID, forumChannel.GuildID)
	if err != nil {
		return nil, err
	}
	slog.Info("threads to analyze", "count", len(threads))

	type threadResult struct {
		name      string
		threadID  string
		reactions map[string][]*discordgo.User
	}

	jobs := make(chan *discordgo.Channel, len(threads))
	results := make(chan threadResult, len(threads))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for thread := range jobs {
				reactions := fetchReactionsWithUsers(b.Session, thread.ID)
				name := guild.ExtractName(thread.Name)
				total := 0
				for _, users := range reactions {
					total += len(users)
				}
				slog.Info("thread reactions fetched", "name", name, "reactions", total)
				results <- threadResult{name: name, threadID: thread.ID, reactions: reactions}
			}
		}()
	}
	for _, t := range threads {
		jobs <- t
	}
	close(jobs)
	go func() { wg.Wait(); close(results) }()

	var threadData []threadResult
	userThreads := make(map[string]map[string]bool)
	allUsers := make(map[string]*discordgo.User)

	botID := b.Session.State.User.ID
	for r := range results {
		threadData = append(threadData, r)
		for _, users := range r.reactions {
			for _, u := range users {
				if u.ID == botID {
					continue
				}
				if userThreads[u.ID] == nil {
					userThreads[u.ID] = make(map[string]bool)
				}
				userThreads[u.ID][r.threadID] = true
				allUsers[u.ID] = u
			}
		}
	}

	// Resolve server display names (Nick > GlobalName > Username) via GuildMember.
	displayNames := resolveDisplayNames(b.Session, forumChannel.GuildID, allUsers)

	counts := make(map[string]int, len(userThreads))
	for uid, ts := range userThreads {
		counts[uid] = len(ts)
	}
	weights := ComputeVoterWeights(counts)

	var records []VoteRecord
	for _, tr := range threadData {
		userEmojis := make(map[string][]string)
		for emoji, users := range tr.reactions {
			for _, u := range users {
				if u.ID == botID {
					continue
				}
				userEmojis[u.ID] = append(userEmojis[u.ID], emoji)
			}
		}
		for uid, emojis := range userEmojis {
			weight := weights[uid]
			var points float64
			for _, emoji := range emojis {
				pts := 0
				switch emoji {
				case "⭐":
					pts = scorePerStar
				case "👍", "👍🏻", "👍🏼", "👍🏽", "👍🏾", "👍🏿", "🔥":
					pts = scorePerLike
				}
				points += float64(pts) * weight
			}
			records = append(records, VoteRecord{
				GuildName:    tr.name,
				ThreadID:     tr.threadID,
				Username:     displayNames[uid],
				UserID:       uid,
				Emojis:       emojis,
				ThreadsVoted: counts[uid],
				Weight:       weight,
				Points:       points,
			})
		}
	}
	return records, nil
}

// resolveDisplayNames fetches guild member display names for all users in parallel.
// Priority: server nickname > global display name > username.
func resolveDisplayNames(s *discordgo.Session, guildID string, users map[string]*discordgo.User) map[string]string {
	type result struct {
		uid  string
		name string
	}

	ch := make(chan result, len(users))
	sem := make(chan struct{}, numWorkers)

	var wg sync.WaitGroup
	for uid, u := range users {
		wg.Add(1)
		go func(uid string, u *discordgo.User) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if m, err := s.GuildMember(guildID, uid); err == nil {
				ch <- result{uid, m.DisplayName()}
				return
			}
			// Fallback to user object already fetched from reactions.
			ch <- result{uid, u.DisplayName()}
		}(uid, u)
	}
	wg.Wait()
	close(ch)

	names := make(map[string]string, len(users))
	for r := range ch {
		names[r.uid] = r.name
	}
	return names
}

// fetchReactionsWithUsers fetches all reactor User objects per emoji in parallel.
func fetchReactionsWithUsers(s *discordgo.Session, threadID string) map[string][]*discordgo.User {
	type result struct {
		emoji string
		users []*discordgo.User
	}

	ch := make(chan result, len(scoredEmojis))
	var wg sync.WaitGroup
	for _, emoji := range scoredEmojis {
		wg.Add(1)
		go func(emoji string) {
			defer wg.Done()
			var users []*discordgo.User
			var after string
			for {
				page, err := s.MessageReactions(threadID, threadID, emoji, 100, "", after)
				if err != nil || len(page) == 0 {
					break
				}
				users = append(users, page...)
				after = page[len(page)-1].ID
				if len(page) < 100 {
					break
				}
			}
			if len(users) > 0 {
				ch <- result{emoji, users}
			}
		}(emoji)
	}
	wg.Wait()
	close(ch)

	reactions := make(map[string][]*discordgo.User)
	for r := range ch {
		reactions[r.emoji] = r.users
	}
	return reactions
}

// EmojiLabel returns a canonical display label for an emoji.
func EmojiLabel(emoji string) string {
	switch emoji {
	case "⭐":
		return "⭐ star"
	case "🔥":
		return "🔥 fire"
	default:
		if strings.HasPrefix(emoji, "👍") {
			return "👍 like"
		}
		return emoji
	}
}
