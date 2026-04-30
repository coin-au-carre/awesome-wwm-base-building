package discord

import (
	"fmt"
	"log/slog"
	"slices"
	"time"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

// WhateverPost represents a single message from #whatever-showcase with images.
type WhateverPost struct {
	ID         string   `json:"id"`
	AuthorName string   `json:"authorName"`
	AuthorID   string   `json:"authorId"`
	Images     []string `json:"images"`
	Reactions  int      `json:"reactions"`
	MessageURL string   `json:"messageUrl"`
	PostedAt   string   `json:"postedAt"`
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

	var posts []WhateverPost
	for _, msg := range all {
		if msg.Author == nil {
			continue
		}
		images := imagesFromMessage(msg)
		if len(images) == 0 {
			continue
		}
		name := msg.Author.GlobalName
		if name == "" {
			name = msg.Author.Username
		}
		posts = append(posts, WhateverPost{
			ID:         msg.ID,
			AuthorName: name,
			AuthorID:   msg.Author.ID,
			Images:     images,
			Reactions:  sumReactions(msg),
			MessageURL: fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID, channelID, msg.ID),
			PostedAt:   msg.Timestamp.Format(time.RFC3339),
		})
	}
	return posts, nil
}

func imagesFromMessage(msg *discordgo.Message) []string {
	var urls []string
	for _, att := range msg.Attachments {
		if guild.IsImage(att.Filename) {
			urls = append(urls, att.URL)
		}
	}
	return urls
}

func sumReactions(msg *discordgo.Message) int {
	n := 0
	for _, r := range msg.Reactions {
		n += r.Count
	}
	return n
}
