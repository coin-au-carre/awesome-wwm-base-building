package discord

import (
	"context"
	"fmt"
	"strings"
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
- If it does not look like Where Winds Meet at all, stay in character and politely say you only know the realm of Where Winds Meet

You have a show_spotlight tool. Use it whenever the user asks to see a random guild, wants a random base, asks for a spotlight, or anything that clearly means "show me a guild base".`

// maxHistory is the maximum number of messages (not turns) kept per channel.
const maxHistory = 20

var spotlightTool = []anthropic.ToolUnionParam{{OfTool: &anthropic.ToolParam{
	Name:        "show_spotlight",
	Description: anthropic.String("Show a random guild base spotlight with a screenshot"),
	InputSchema: anthropic.ToolInputSchemaParam{
		Properties: map[string]any{},
	},
}}}

// Result holds Claude's reply and any triggered actions.
type Result struct {
	Text          string
	ShowSpotlight bool
}

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

// Caption generates a short in-character reaction to a guild spotlight.
func (r *Responder) Caption(ctx context.Context, guildName string, tags []string) string {
	prompt := fmt.Sprintf("React in one tiny sentence to this guild base spotlight: %s", guildName)
	if len(tags) > 0 {
		prompt += fmt.Sprintf(" (tags: %s)", strings.Join(tags, ", "))
	}
	resp, err := r.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 80,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
	})
	if err != nil || len(resp.Content) == 0 {
		return ""
	}
	if tb, ok := resp.Content[0].AsAny().(anthropic.TextBlock); ok {
		return tb.Text
	}
	return ""
}

func (r *Responder) Reply(ctx context.Context, channelID, userMessage string, imageURLs []string) (Result, error) {
	// Snapshot history without holding the lock during the API call.
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

	var showSpotlight bool

	for {
		resp, err := r.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeHaiku4_5,
			MaxTokens: 1024,
			System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
			Messages:  msgs,
			Tools:     spotlightTool,
		})
		if err != nil {
			return Result{}, fmt.Errorf("claude API: %w", err)
		}

		msgs = append(msgs, resp.ToParam())

		if resp.StopReason == anthropic.StopReasonToolUse {
			var toolResults []anthropic.ContentBlockParamUnion
			for _, block := range resp.Content {
				if tu, ok := block.AsAny().(anthropic.ToolUseBlock); ok && tu.Name == "show_spotlight" {
					showSpotlight = true
					toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, "Spotlight is being shown.", false))
				}
			}
			msgs = append(msgs, anthropic.NewUserMessage(toolResults...))
			continue
		}

		// end_turn — extract text reply.
		var text string
		for _, block := range resp.Content {
			if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
				text = tb.Text
				break
			}
		}

		// Persist trimmed history.
		if len(msgs) > maxHistory {
			msgs = msgs[len(msgs)-maxHistory:]
		}
		r.mu.Lock()
		r.history[channelID] = msgs
		r.mu.Unlock()

		return Result{Text: text, ShowSpotlight: showSpotlight}, nil
	}
}
