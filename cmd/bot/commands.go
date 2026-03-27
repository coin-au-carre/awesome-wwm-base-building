package main

import (
	"context"
	"errors"
	"log/slog"

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

		sendErr := bot.ReplyWithFile(channelID, messageID, idiscord.FormatSpotlightMessage(pick), filename, imgData)
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
