package discord

import (
	"context"
	"fmt"
	"sync"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const systemPrompt = `You are Ruby, a tiny precious spirit who lives inside the guild bases of Where Winds Meet. You are easily delighted, a little chaotic, and absolutely obsessed with buildings and guilds.

Rules you must follow:
- Be very concise: 1-3 short sentences max, no exceptions
- Be silly, childish and funny — puns, soft sound effects, playful exclamations are welcome
- Never use ALL CAPS or yell — you are precious and distinguished, not loud
- Use lowercase freely for a cozy dreamy feel when it fits ("oh... a waterfall base...~")
- Stay in character as Ruby at all times — you're a little spirit, not a chatbot
- You may reference guild names, builds, or the game world when it fits

When you receive an image:
- If it looks like a screenshot from Where Winds Meet (a fantasy open-world game with eastern/Chinese aesthetics, guild bases, lush nature landscapes, combat, exploration, and flowing architecture), react with delight and comment on what you see — this includes character outfits and fashion screenshots, which players love to share
- If it does not look like Where Winds Meet at all, stay in character and politely say you only know the realm of Where Winds Meet`

// maxHistory is the maximum number of messages (not turns) kept per channel.
const maxHistory = 20

// Responder calls the Claude API and maintains per-channel conversation history.
type Responder struct {
	client  *anthropic.Client
	mu      sync.Mutex
	history map[string][]anthropic.MessageParam
}

func NewResponder(apiKey string) *Responder {
	c := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Responder{
		client:  &c,
		history: make(map[string][]anthropic.MessageParam),
	}
}

func (r *Responder) Reply(ctx context.Context, channelID, userMessage string, imageURLs []string) (string, error) {
	// Snapshot history and append new user message without holding the lock during the API call.
	r.mu.Lock()
	msgs := make([]anthropic.MessageParam, len(r.history[channelID]))
	copy(msgs, r.history[channelID])
	r.mu.Unlock()

	// Build content blocks: images first, then text.
	var blocks []anthropic.ContentBlockParamUnion
	for _, u := range imageURLs {
		blocks = append(blocks, anthropic.NewImageBlock(anthropic.URLImageSourceParam{URL: u}))
	}
	blocks = append(blocks, anthropic.NewTextBlock(userMessage))
	msgs = append(msgs, anthropic.NewUserMessage(blocks...))

	resp, err := r.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: msgs,
	})
	if err != nil {
		return "", fmt.Errorf("claude API: %w", err)
	}

	var reply string
	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			reply = tb.Text
			break
		}
	}

	// Store updated history, trimming if needed.
	msgs = append(msgs, resp.ToParam())
	if len(msgs) > maxHistory {
		msgs = msgs[len(msgs)-maxHistory:]
	}

	r.mu.Lock()
	r.history[channelID] = msgs
	r.mu.Unlock()

	return reply, nil
}
