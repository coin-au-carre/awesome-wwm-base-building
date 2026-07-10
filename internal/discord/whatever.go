package discord

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
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
	start := time.Now()
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

	slog.Info("fetched messages", "total", len(all), "channel", channelID, "elapsed", time.Since(start))

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

	result := make([]WhateverPost, 0, len(candidates))
	for _, c := range candidates {
		reactions := maxReactionCount(c.msg)
		if reactions == 0 {
			continue
		}
		name := c.msg.Author.GlobalName
		if name == "" {
			name = c.msg.Author.Username
		}
		result = append(result, WhateverPost{
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
		})
	}
	slog.Info("whatever showcase fetched", "posts", len(result), "elapsed", time.Since(start))
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

// maxReactionCount approximates unique reactor count as the single
// highest-count emoji on the message, with zero extra API calls (unlike
// fetching each emoji's reactor list to dedupe exactly). This undercounts a
// user who reacted with multiple emoji as if they were one reactor across
// the whole message, but never overcounts the way summing all emoji counts
// would when someone stacks ⭐👍🔥 on their own post.
func maxReactionCount(msg *discordgo.Message) int {
	max := 0
	for _, r := range msg.Reactions {
		if r.Count > max {
			max = r.Count
		}
	}
	return max
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
