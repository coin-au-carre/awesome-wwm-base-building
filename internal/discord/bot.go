// internal/discord/bot.go
package discord

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

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
		discordgo.IntentsMessageContent |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsGuildInvites

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
	_, err := b.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:   msg,
		Reference: &discordgo.MessageReference{MessageID: messageID},
		Flags:     discordgo.MessageFlagsSuppressEmbeds,
	})
	if err != nil {
		slog.Warn("failed to send reply", "err", err)
	}
}

// ReplyChunked splits msg at paragraph/sentence boundaries and sends each chunk
// as a reply, staying within Discord's 2000-character limit.
func (b *Bot) ReplyChunked(channelID, messageID, msg string) {
	for i, chunk := range splitMessage(msg, 2000) {
		ref := messageID
		if i > 0 {
			ref = ""
		}
		var err error
		if ref != "" {
			_, err = b.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
				Content:   chunk,
				Reference: &discordgo.MessageReference{MessageID: ref},
				Flags:     discordgo.MessageFlagsSuppressEmbeds,
			})
		} else {
			_, err = b.Session.ChannelMessageSend(channelID, chunk)
		}
		if err != nil {
			slog.Warn("failed to send reply chunk", "chunk", i, "err", err)
		}
	}
}

// splitMessage splits text into chunks of at most maxLen characters,
// breaking at paragraph boundaries first, then sentence ends, then words.
func splitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= maxLen {
			chunks = append(chunks, text)
			break
		}
		cut := findSplit(text, maxLen)
		chunks = append(chunks, strings.TrimSpace(text[:cut]))
		text = strings.TrimSpace(text[cut:])
	}
	return chunks
}

// findSplit returns the index at which to cut text, preferring paragraph > sentence > word boundaries.
func findSplit(text string, maxLen int) int {
	window := text[:maxLen]

	if i := strings.LastIndex(window, "\n\n"); i > 0 {
		return i + 2
	}
	if i := strings.LastIndex(window, "\n"); i > 0 {
		return i + 1
	}
	for _, sep := range []string{". ", "! ", "? "} {
		if i := strings.LastIndex(window, sep); i > 0 {
			return i + len(sep)
		}
	}
	if i := strings.LastIndex(window, " "); i > 0 {
		return i + 1
	}
	return maxLen
}

func (b *Bot) ReplyWithEmbeds(channelID, messageID, msg string) {
	_, err := b.Session.ChannelMessageSendReply(channelID, msg, &discordgo.MessageReference{MessageID: messageID})
	if err != nil {
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
		slog.Warn("failed to send message", "channel", channelID, "err", err)
	}
}

func (b *Bot) SendReturnID(channelID, msg string) string {
	m, err := b.Session.ChannelMessageSend(channelID, msg)
	if err != nil {
		slog.Warn("failed to send message", "err", err)
		return ""
	}
	return m.ID
}

func (b *Bot) EditMessage(channelID, messageID, msg string) {
	if _, err := b.Session.ChannelMessageEdit(channelID, messageID, msg); err != nil {
		slog.Warn("failed to edit message", "err", err)
	}
}

func (b *Bot) SendWithFile(channelID, content, filename string, file io.Reader) error {
	_, err := b.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: content,
		Files:   []*discordgo.File{{Name: filename, Reader: file}},
		Flags:   discordgo.MessageFlagsSuppressEmbeds,
	})
	if err != nil {
		slog.Warn("failed to send file message", "channel", channelID, "err", err)
	}
	return err
}

func (b *Bot) NotifyWithFile(content, filename string, file io.Reader) error {
	if b.channelID == "" {
		slog.Warn("no channel ID configured, skipping notification")
		return nil
	}
	_, err := b.Session.ChannelMessageSendComplex(b.channelID, &discordgo.MessageSend{
		Content: content,
		Files:   []*discordgo.File{{Name: filename, Reader: file}},
		Flags:   discordgo.MessageFlagsSuppressEmbeds,
	})
	if err != nil {
		slog.Warn("failed to send file notification", "err", err)
	}
	return err
}

func (b *Bot) NotifyIf(cond bool, msg string) {
	if cond {
		b.Notify(msg)
	} else {
		slog.Info("notification suppressed (no-notify)", "msg", msg)
	}
}
