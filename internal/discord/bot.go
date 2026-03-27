// internal/discord/bot.go
package discord

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session   *discordgo.Session
	channelID string
}

func NewBot(token, channelID string) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}
	s.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent

	return &Bot{Session: s, channelID: channelID}, nil
}

func (b *Bot) Open() error {
	return b.Session.Open()
}

func (b *Bot) Close() {
	if err := b.Session.Close(); err != nil {
		slog.Warn("closing session", "err", err)
	}
}

func (b *Bot) Notify(msg string) {
	if b.channelID == "" {
		slog.Warn("no channel ID configured, skipping notification")
		return
	}
	if _, err := b.Session.ChannelMessageSend(b.channelID, msg); err != nil {
		slog.Warn("failed to send notification", "err", err)
	}
}

func (b *Bot) Reply(channelID, messageID, msg string) {
	if _, err := b.Session.ChannelMessageSendReply(channelID, msg, &discordgo.MessageReference{MessageID: messageID}); err != nil {
		slog.Warn("failed to send reply", "err", err)
	}
}

func (b *Bot) ReplyWithFile(channelID, messageID, content, filename string, file io.Reader) error {
	_, err := b.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:   content,
		Files:     []*discordgo.File{{Name: filename, Reader: file}},
		Reference: &discordgo.MessageReference{MessageID: messageID},
		Flags:     discordgo.MessageFlagsSuppressEmbeds,
	})
	if err != nil {
		slog.Warn("failed to send file reply", "err", err)
	}
	return err
}

func (b *Bot) Send(channelID, msg string) {
	if _, err := b.Session.ChannelMessageSend(channelID, msg); err != nil {
		slog.Warn("failed to send message", "err", err)
	}
}

func (b *Bot) NotifyIf(cond bool, msg string) {
	if cond {
		b.Notify(msg)
	} else {
		slog.Info("notification suppressed (no-notify)", "msg", msg)
	}
}
