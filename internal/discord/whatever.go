package discord

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

// ReactionDetail holds a single emoji reaction and its count.
type ReactionDetail struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
}

// WhateverPost represents a single message from #whatever-showcase with images or videos.
type WhateverPost struct {
	ID              string           `json:"id"`
	AuthorName      string           `json:"authorName"`
	AuthorID        string           `json:"authorId"`
	Content         string           `json:"content,omitempty"`
	Images          []string         `json:"images"`
	Videos          []string         `json:"videos,omitempty"`
	Reactions       int              `json:"reactions"`
	ReactionDetails []ReactionDetail `json:"reactionDetails"`
	MessageURL      string           `json:"messageUrl"`
	PostedAt        string           `json:"postedAt"`
}

// FetchWhateverShowcase fetches all image-bearing messages from the given plain channel.
func FetchWhateverShowcase(s *discordgo.Session, channelID, guildID string) ([]WhateverPost, error) {
	var all []*discordgo.Message
	var before string
	for {
		msgs, err := s.ChannelMessages(channelID, 100, before, "", "")
		if err != nil {
			return nil, fmt.Errorf("fetching messages: %w", err)
		}
		if len(msgs) == 0 {
			break
		}
		all = append(all, msgs...)
		before = msgs[len(msgs)-1].ID
		if len(msgs) < 100 {
			break
		}
	}

	slog.Info("fetched messages", "total", len(all), "channel", channelID)

	// Discord returns newest-first; reverse to chronological order.
	slices.Reverse(all)

	type candidate struct {
		msg    *discordgo.Message
		images []string
		videos []string
	}
	var candidates []candidate
	for _, msg := range all {
		if msg.Author == nil {
			continue
		}
		images := imagesFromMessage(msg)
		videos := videosFromMessage(msg)
		if len(images) == 0 && len(videos) == 0 {
			continue
		}
		if hasReaction(msg, "🚫") {
			continue
		}
		candidates = append(candidates, candidate{msg: msg, images: images, videos: videos})
	}

	// Fetching unique reactor counts is one Discord API call per emoji per
	// message; run it through a worker pool instead of serially or it takes
	// minutes on a busy channel.
	posts := make([]WhateverPost, len(candidates))
	keep := make([]bool, len(candidates))
	jobs := make(chan int, len(candidates))
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				c := candidates[i]
				reactions := uniqueReactorCount(s, channelID, c.msg)
				if reactions == 0 {
					continue
				}
				name := c.msg.Author.GlobalName
				if name == "" {
					name = c.msg.Author.Username
				}
				posts[i] = WhateverPost{
					ID:              c.msg.ID,
					AuthorName:      name,
					AuthorID:        c.msg.Author.ID,
					Content:         strings.TrimSpace(c.msg.Content),
					Images:          c.images,
					Videos:          c.videos,
					Reactions:       reactions,
					ReactionDetails: reactionDetails(c.msg),
					MessageURL:      fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID, channelID, c.msg.ID),
					PostedAt:        c.msg.Timestamp.Format(time.RFC3339),
				}
				keep[i] = true
			}
		}()
	}
	for i := range candidates {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	result := make([]WhateverPost, 0, len(posts))
	for i, p := range posts {
		if keep[i] {
			result = append(result, p)
		}
	}
	return result, nil
}

func imagesFromMessage(msg *discordgo.Message) []string {
	urls := []string{}
	for _, att := range msg.Attachments {
		if guild.IsImage(att.Filename) {
			urls = append(urls, att.URL)
		}
	}
	return urls
}

func videosFromMessage(msg *discordgo.Message) []string {
	urls := []string{}
	for _, att := range msg.Attachments {
		if guild.IsVideo(att.Filename) {
			urls = append(urls, att.URL)
		}
	}
	return urls
}

// uniqueReactorCount counts distinct users who reacted, regardless of how many
// different emoji each one used, so one person spamming ⭐👍🔥 on the same
// image only counts once.
func uniqueReactorCount(s *discordgo.Session, channelID string, msg *discordgo.Message) int {
	seen := make(map[string]struct{})
	for _, r := range msg.Reactions {
		users, err := s.MessageReactions(channelID, msg.ID, r.Emoji.APIName(), 100, "", "")
		if err != nil {
			slog.Warn("fetching reaction users", "error", err, "message", msg.ID, "emoji", r.Emoji.Name)
			continue
		}
		for _, u := range users {
			seen[u.ID] = struct{}{}
		}
	}
	return len(seen)
}

func hasReaction(msg *discordgo.Message, emoji string) bool {
	for _, r := range msg.Reactions {
		if r.Emoji.Name == emoji {
			return true
		}
	}
	return false
}

func reactionDetails(msg *discordgo.Message) []ReactionDetail {
	var details []ReactionDetail
	for _, r := range msg.Reactions {
		emoji := r.Emoji.Name
		if emoji == "" {
			continue
		}
		details = append(details, ReactionDetail{Emoji: emoji, Count: r.Count})
	}
	return details
}
