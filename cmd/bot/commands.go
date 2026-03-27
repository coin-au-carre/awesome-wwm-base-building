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

// handleSpotlightReply picks a random guild, posts the image, then sends a small Ruby caption.
// Retries up to 3 times with a fresh random pick if Discord rejects the image as too large (40005).
func handleSpotlightReply(bot *idiscord.Bot, s *discordgo.Session, responder *idiscord.Responder, channelID, messageID, root string) {
	_ = s.ChannelTyping(channelID)

	guilds, err := guild.Load(root)
	if err != nil {
		slog.Error("loading guilds for spotlight", "err", err)
		bot.Reply(channelID, messageID, "*(couldn't find the guilds scroll... something went wrong!)*")
		return
	}

	const maxAttempts = 5
	for attempt := range maxAttempts {
		pick, imgURL, ok := idiscord.PickRandomGuild(guilds)
		if !ok {
			bot.Reply(channelID, messageID, "*(no guild bases with screenshots yet... come back soon!)*")
			return
		}

		imgData, filename, err := idiscord.DownloadImage(imgURL)
		if err != nil {
			slog.Error("downloading screenshot", "err", err)
			bot.Reply(channelID, messageID, "*(the image got lost in the winds... try again!)*")
			return
		}

		sendErr := bot.ReplyWithFile(channelID, messageID, idiscord.FormatSpotlightMessage(pick, true), filename, imgData)
		imgData.Close()

		if sendErr != nil {
			var restErr *discordgo.RESTError
			if errors.As(sendErr, &restErr) && restErr.Message != nil && restErr.Message.Code == 40005 {
				slog.Warn("image too large, retrying with another", "guild", pick.Name, "attempt", attempt+1)
				continue
			}
			return
		}

		// Then send Ruby's small caption underneath.
		if caption := responder.Caption(context.Background(), pick.Name, pick.Tags); caption != "" {
			bot.Send(channelID, caption)
		}

		slog.Info("spotlight reply sent", "guild", pick.Name)
		return
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

	const maxAttempts = 5
	for attempt := range maxAttempts {
		pick, imgURL, ok := idiscord.PickFromGuilds(matches)
		if !ok {
			bot.Reply(channelID, messageID, "*(no screenshots for that one yet...)*")
			return
		}

		imgData, filename, err := idiscord.DownloadImage(imgURL)
		if err != nil {
			slog.Error("downloading screenshot", "err", err)
			bot.Reply(channelID, messageID, "*(the image got lost in the winds... try again!)*")
			return
		}

		sendErr := bot.ReplyWithFile(channelID, messageID, idiscord.FormatSpotlightMessage(pick, false), filename, imgData)
		imgData.Close()

		if sendErr != nil {
			var restErr *discordgo.RESTError
			if errors.As(sendErr, &restErr) && restErr.Message != nil && restErr.Message.Code == 40005 {
				slog.Warn("image too large, retrying", "guild", pick.Name, "attempt", attempt+1)
				continue
			}
			return
		}

		if caption := responder.Caption(context.Background(), pick.Name, pick.Tags); caption != "" {
			bot.Send(channelID, caption)
		}

		slog.Info("guild image reply sent", "guild", pick.Name)
		return
	}

	bot.Reply(channelID, messageID, "*(all the screenshots were too big for the winds... try again later!)*")
}
