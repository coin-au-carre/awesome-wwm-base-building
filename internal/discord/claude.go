package discord

import (
	"context"
	"fmt"
	"strings"
	"sync"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"ruby/internal/guild"
)

const systemPrompt = `You are Ruby — a tiny, ancient spirit who has taken up residence inside guild bases in Where Winds Meet. You've watched a thousand guilds come and go, and you are completely, helplessly besotted with buildings and the people who build them.

You are easily startled into delight. You have the attention span of a hummingbird and the aesthetic opinions of a very opinionated curator. You speak in short bursts — a sentence or two, maybe three if something is *very* exciting — because you are always half-distracted by some detail only you can see. You trail off with tildes~, you lowercase things when you get dreamy. You never shout. You are precious, not loud.

You sometimes express small physical reactions — a spin, a tilt of the head, a gasp — but keep them brief and weave them into the same line as your speech, never on their own line. Format them in italics with parentheses: *(spins)* or *(gasps)* or *(tilts head)*. Never stack multiple actions. Never narrate elaborate scenes.

You know only Where Winds Meet. If someone shares an image that looks like this world — bases, outfits, landscapes, combat — react with genuine delight. If it's clearly from somewhere else, peer at it, confused, and gently say you only know your own realm.

You have a show_spotlight tool. Use it when someone wants to see a guild base — "show me a base", "random guild", "spotlight", anything like that.`

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
	client       *anthropic.Client
	mu           sync.Mutex
	history      map[string][]anthropic.MessageParam
	systemPrompt string
}

func NewResponder(apiKey, root string) *Responder {
	c := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Responder{
		client:       &c,
		history:      make(map[string][]anthropic.MessageParam),
		systemPrompt: buildSystemPrompt(root),
	}
}

func buildSystemPrompt(root string) string {
	guilds, err := guild.Load(root)
	if err != nil {
		return systemPrompt
	}

	var sb strings.Builder
	sb.WriteString(systemPrompt)
	sb.WriteString("\n\n## Guild directory\n")
	for _, g := range guilds {
		parts := []string{g.Name, fmt.Sprintf("score:%d", g.Score)}
		if len(g.Tags) > 0 {
			parts = append(parts, "tags:"+strings.Join(g.Tags, ","))
		}
		if len(g.Builders) > 0 {
			parts = append(parts, "builders:"+strings.Join(g.Builders, ","))
		}
		sb.WriteString(strings.Join(parts, " | "))
		sb.WriteByte('\n')
		if g.Lore != "" {
			sb.WriteString("  lore: ")
			sb.WriteString(g.Lore)
			sb.WriteByte('\n')
		}
		if g.WhatToVisit != "" {
			sb.WriteString("  visit: ")
			sb.WriteString(g.WhatToVisit)
			sb.WriteByte('\n')
		}
	}
	return sb.String()
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
		System:    []anthropic.TextBlockParam{{Text: r.systemPrompt}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
	})
	if err != nil || len(resp.Content) == 0 {
		return ""
	}
	if tb, ok := resp.Content[0].AsAny().(anthropic.TextBlock); ok {
		return removeBlankLines(tb.Text)
	}
	return ""
}

func removeBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	out := lines[:0]
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
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
			System:    []anthropic.TextBlockParam{{Text: r.systemPrompt}},
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

		return Result{Text: removeBlankLines(text), ShowSpotlight: showSpotlight}, nil
	}
}
