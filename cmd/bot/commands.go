package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
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

const maxCatalogImages = 4

// handleCatalogItemsReply searches the catalog for items matching query, sends up to
// maxCatalogImages images as Discord file attachments, and appends a website link.
func handleCatalogItemsReply(bot *idiscord.Bot, s *discordgo.Session, channelID, messageID, root, query string) {
	_ = s.ChannelTyping(channelID)

	items := idiscord.SearchCatalogItems(root, query, maxCatalogImages)
	if len(items) == 0 {
		bot.Reply(channelID, messageID, fmt.Sprintf("*(I searched everywhere but couldn't find any items matching \"%s\"...)*", query))
		return
	}

	websiteURL := fmt.Sprintf("https://www.wherebuildersmeet.com/catalog/?q=%%23%s", url.QueryEscape(query))

	var files []*discordgo.File
	for _, item := range items {
		f, err := os.Open(idiscord.CatalogImagePath(root, item))
		if err != nil {
			slog.Warn("opening catalog image", "file", item.Filename, "err", err)
			continue
		}
		files = append(files, &discordgo.File{Name: item.Name + filepath.Ext(item.Filename), Reader: f})
	}
	defer func() {
		for _, f := range files {
			if rc, ok := f.Reader.(*os.File); ok {
				rc.Close()
			}
		}
	}()

	if len(files) == 0 {
		bot.Reply(channelID, messageID, fmt.Sprintf("*(found some items but couldn't open their images... see the full list: %s)*", websiteURL))
		return
	}

	content := fmt.Sprintf("[Browse all **%s** items on the website](%s)", query, websiteURL)
	_, err := bot.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:   content,
		Files:     files,
		Reference: &discordgo.MessageReference{MessageID: messageID},
		Flags:     discordgo.MessageFlagsSuppressEmbeds,
	})
	if err != nil {
		slog.Warn("sending catalog images", "err", err)
	}
}

// handleSoloSpotlightReply picks a random solo build and posts its image.
func handleSoloSpotlightReply(bot *idiscord.Bot, s *discordgo.Session, responder *idiscord.Responder, channelID, messageID, root string) {
	_ = s.ChannelTyping(channelID)

	solos, err := guild.LoadFile(filepath.Join(root, "data", "solos.json"))
	if err != nil {
		slog.Error("loading solos for spotlight", "err", err)
		bot.Reply(channelID, messageID, "*(couldn't find the solo builds scroll... something went wrong!)*")
		return
	}

	for range maxImageAttempts {
		pick, imgURL, ok := idiscord.PickRandomGuild(solos)
		if !ok {
			bot.Reply(channelID, messageID, "*(no solo builds with screenshots yet... come back soon!)*")
			return
		}
		if sent, fatal := sendGuildImage(bot, responder, channelID, messageID, pick, imgURL, true); sent || fatal {
			if sent {
				slog.Info("solo spotlight reply sent", "build", pick.Name)
			}
			return
		}
		slog.Warn("solo image too large, retrying", "build", pick.Name)
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
