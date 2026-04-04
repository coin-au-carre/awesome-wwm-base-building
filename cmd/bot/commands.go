package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	idiscord "ruby/internal/discord"
	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

const maxImageAttempts = 5

// sendGuildImage downloads the image for pick and sends it with a caption.
// Returns true on success, false if the image was too large (caller should retry with a different pick).
func sendGuildImage(bot *idiscord.Bot, responder *idiscord.Responder, channelID, messageID string, pick guild.Guild, imgURL string, random bool) (ok bool, fatal bool) {
	imgData, filename, err := idiscord.DownloadImage(imgURL)
	if err != nil {
		slog.Error("downloading screenshot", "err", err)
		bot.Reply(channelID, messageID, "*(the image got lost in the winds... try again!)*")
		return false, true
	}

	sendErr := bot.ReplyWithFile(channelID, messageID, idiscord.FormatSpotlightMessage(pick, random), filename, imgData)
	imgData.Close()

	if sendErr != nil {
		var restErr *discordgo.RESTError
		if errors.As(sendErr, &restErr) && restErr.Message != nil && restErr.Message.Code == 40005 {
			return false, false // too large — retry
		}
		return false, true
	}

	if caption := responder.Caption(context.Background(), pick.Name, pick.Tags); caption != "" {
		bot.Send(channelID, caption)
	}
	return true, false
}

// handleSpotlightReply picks a random guild, posts the image, then sends a small Ruby caption.
// Retries up to maxImageAttempts times if Discord rejects the image as too large (40005).
func handleSpotlightReply(bot *idiscord.Bot, s *discordgo.Session, responder *idiscord.Responder, channelID, messageID, root string) {
	_ = s.ChannelTyping(channelID)

	guilds, err := guild.Load(root)
	if err != nil {
		slog.Error("loading guilds for spotlight", "err", err)
		bot.Reply(channelID, messageID, "*(couldn't find the guilds scroll... something went wrong!)*")
		return
	}

	for range maxImageAttempts {
		pick, imgURL, ok := idiscord.PickRandomGuild(guilds)
		if !ok {
			bot.Reply(channelID, messageID, "*(no guild bases with screenshots yet... come back soon!)*")
			return
		}
		if sent, fatal := sendGuildImage(bot, responder, channelID, messageID, pick, imgURL, true); sent || fatal {
			if sent {
				slog.Info("spotlight reply sent", "guild", pick.Name)
			}
			return
		}
		slog.Warn("image too large, retrying with another", "guild", pick.Name)
	}

	bot.Reply(channelID, messageID, "*(all the screenshots were too big for the winds... try again later!)*")
}

// handleGuildImageReply searches for a guild matching query and posts one of its screenshots.
func handleGuildImageReply(bot *idiscord.Bot, s *discordgo.Session, responder *idiscord.Responder, channelID, messageID, root, query string) {
	_ = s.ChannelTyping(channelID)

	guilds, err := guild.Load(root)
	if err != nil {
		slog.Error("loading guilds for guild image", "err", err)
		bot.Reply(channelID, messageID, "*(couldn't find the guilds scroll... something went wrong!)*")
		return
	}

	q := strings.ToLower(query)
	var matches []guild.Guild
	for _, g := range guilds {
		if len(g.Screenshots) > 0 && strings.Contains(strings.ToLower(g.Name), q) {
			matches = append(matches, g)
		}
	}
	if len(matches) == 0 {
		bot.Reply(channelID, messageID, "*(I couldn't find a base by that name... are you sure it's in the guilds scroll?)*")
		return
	}

	for range maxImageAttempts {
		pick, imgURL, ok := idiscord.PickFromGuilds(matches)
		if !ok {
			bot.Reply(channelID, messageID, "*(no screenshots for that one yet...)*")
			return
		}
		if sent, fatal := sendGuildImage(bot, responder, channelID, messageID, pick, imgURL, false); sent || fatal {
			if sent {
				slog.Info("guild image reply sent", "guild", pick.Name)
			}
			return
		}
		slog.Warn("image too large, retrying", "guild", pick.Name)
	}

	bot.Reply(channelID, messageID, "*(all the screenshots were too big for the winds... try again later!)*")
}
