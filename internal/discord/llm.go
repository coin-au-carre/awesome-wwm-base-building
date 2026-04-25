package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"

	anthropic "github.com/anthropics/anthropic-sdk-go"
)

// LLMResponder is the interface that all responder types (Claude API, CLI, Ollama) implement.
type LLMResponder interface {
	Reply(ctx context.Context, channelID, userMessage string, imageURLs []string) (Result, error)
	Caption(ctx context.Context, guildName string, tags []string) string
}

// Result holds Claude's reply and any triggered actions.
type Result struct {
	Text            string
	ShowSpotlight   bool
	ShowSolo        bool
	GuildImageQuery string
	CatalogQuery    string
}

// Responder calls the Claude API and maintains per-channel conversation history.
type Responder struct {
	client       *anthropic.Client
	mu           sync.Mutex
	history      map[string][]anthropic.MessageParam // API mode
	sessions     map[string]string                   // CLI mode: channelID → session ID
	systemPrompt string
}

// maxHistory is the maximum number of messages (not turns) kept per channel.
const maxHistory = 20

var tools = []anthropic.ToolUnionParam{
	{OfTool: &anthropic.ToolParam{
		Name:        "show_spotlight",
		Description: anthropic.String("Show a random guild base spotlight with a screenshot"),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{},
		},
	}},
	{OfTool: &anthropic.ToolParam{
		Name:        "show_solo_spotlight",
		Description: anthropic.String("Show a random solo build spotlight with a screenshot"),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{},
		},
	}},
	{OfTool: &anthropic.ToolParam{
		Name:        "show_guild_image",
		Description: anthropic.String("Show an image/screenshot of a specific guild base by name. Use when someone asks to see an image of a named guild or base."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The guild name or base name to search for",
				},
			},
			Required: []string{"query"},
		},
	}},
	{OfTool: &anthropic.ToolParam{
		Name:        "show_catalog_items",
		Description: anthropic.String("Show images of building items from the catalog. Use when someone asks to see or browse specific building pieces (e.g. 'show me carpets', 'what do walls look like'). A few images will be displayed alongside a link to the full catalog."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search term matching item names, subcategory (e.g. 'Floor', 'Wall', 'Roof'), or a keyword like 'carpet'",
				},
			},
			Required: []string{"query"},
		},
	}},
}

func NewResponder(client *anthropic.Client, root string) *Responder {
	return &Responder{
		client:       client,
		history:      make(map[string][]anthropic.MessageParam),
		systemPrompt: buildSystemPrompt(root),
	}
}

func NewCLIResponder(root string) *Responder {
	return &Responder{
		sessions:     make(map[string]string),
		systemPrompt: buildSystemPrompt(root),
	}
}

// Caption generates a short in-character reaction to a guild spotlight.
func (r *Responder) Caption(ctx context.Context, guildName string, tags []string) string {
	prompt := fmt.Sprintf("React in one tiny sentence to this guild base spotlight: %s", guildName)
	if len(tags) > 0 {
		prompt += fmt.Sprintf(" (tags: %s)", strings.Join(tags, ", "))
	}
	if r.client == nil {
		result, err := runCLI(ctx, r.systemPrompt, "", prompt)
		if err != nil {
			return ""
		}
		return removeBlankLines(result.text)
	}
	resp, err := r.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 100,
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

func (r *Responder) Reply(ctx context.Context, channelID, userMessage string, imageURLs []string) (Result, error) {
	if r.client == nil {
		return r.replyViaCLI(ctx, channelID, userMessage)
	}

	r.mu.Lock()
	msgs := make([]anthropic.MessageParam, len(r.history[channelID]))
	copy(msgs, r.history[channelID])
	r.mu.Unlock()

	var blocks []anthropic.ContentBlockParamUnion
	for _, u := range imageURLs {
		blocks = append(blocks, anthropic.NewImageBlock(anthropic.URLImageSourceParam{URL: u}))
	}
	blocks = append(blocks, anthropic.NewTextBlock(userMessage))
	msgs = append(msgs, anthropic.NewUserMessage(blocks...))

	var showSpotlight, showSolo bool
	var guildImageQuery, catalogQuery string

	for {
		resp, err := r.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeSonnet4_6,
			MaxTokens: 2048,
			System:    []anthropic.TextBlockParam{{Text: r.systemPrompt}},
			Messages:  msgs,
			Tools:     tools,
		})
		if err != nil {
			return Result{}, fmt.Errorf("claude API: %w", err)
		}

		msgs = append(msgs, resp.ToParam())

		if resp.StopReason == anthropic.StopReasonToolUse {
			var toolResults []anthropic.ContentBlockParamUnion
			for _, block := range resp.Content {
				tu, ok := block.AsAny().(anthropic.ToolUseBlock)
				if !ok {
					continue
				}
				switch tu.Name {
				case "show_spotlight":
					slog.Info("ruby tool: show_spotlight")
					showSpotlight = true
					toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, "Spotlight is being shown.", false))
				case "show_solo_spotlight":
					slog.Info("ruby tool: show_solo_spotlight")
					showSolo = true
					toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, "Solo spotlight is being shown.", false))
				case "show_guild_image":
					var input struct {
						Query string `json:"query"`
					}
					if err := json.Unmarshal(tu.Input, &input); err == nil {
						guildImageQuery = input.Query
					}
					slog.Info("ruby tool: show_guild_image", "query", guildImageQuery)
					toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, "Guild image is being shown.", false))
				case "show_catalog_items":
					var input struct {
						Query string `json:"query"`
					}
					if err := json.Unmarshal(tu.Input, &input); err == nil {
						catalogQuery = input.Query
					}
					slog.Info("ruby tool: show_catalog_items", "query", catalogQuery)
					toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, "Images are being sent to Discord now. Do not list item names or filenames in your reply — just react briefly in character.", false))
				default:
					slog.Warn("ruby tool: unknown", "name", tu.Name)
				}
			}
			msgs = append(msgs, anthropic.NewUserMessage(toolResults...))
			continue
		}

		var text string
		for _, block := range resp.Content {
			if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
				text = tb.Text
				break
			}
		}

		if len(msgs) > maxHistory {
			msgs = msgs[len(msgs)-maxHistory:]
		}
		r.mu.Lock()
		r.history[channelID] = msgs
		r.mu.Unlock()

		return Result{Text: removeBlankLines(text), ShowSpotlight: showSpotlight, ShowSolo: showSolo, GuildImageQuery: guildImageQuery, CatalogQuery: catalogQuery}, nil
	}
}

func (r *Responder) replyViaCLI(ctx context.Context, channelID, userMessage string) (Result, error) {
	r.mu.Lock()
	sessionID := r.sessions[channelID]
	r.mu.Unlock()

	result, err := runCLI(ctx, r.systemPrompt, sessionID, userMessage)
	if err != nil {
		return Result{}, fmt.Errorf("claude CLI: %w", err)
	}

	r.mu.Lock()
	r.sessions[channelID] = result.sessionID
	r.mu.Unlock()

	return parseCLIResult(result.text), nil
}

func parseCLIResult(text string) Result {
	var res Result
	var lines []string
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[SPOTLIGHT]" {
			res.ShowSpotlight = true
			continue
		}
		if trimmed == "[SOLO]" {
			res.ShowSolo = true
			continue
		}
		if strings.HasPrefix(trimmed, "[GUILD:") && strings.HasSuffix(trimmed, "]") {
			res.GuildImageQuery = strings.TrimSuffix(strings.TrimPrefix(trimmed, "[GUILD:"), "]")
			continue
		}
		if strings.HasPrefix(trimmed, "[CATALOG:") && strings.HasSuffix(trimmed, "]") {
			res.CatalogQuery = strings.TrimSuffix(strings.TrimPrefix(trimmed, "[CATALOG:"), "]")
			continue
		}
		lines = append(lines, line)
	}
	res.Text = removeBlankLines(strings.Join(lines, "\n"))
	return res
}

type cliResult struct {
	text      string
	sessionID string
}

func runCLI(ctx context.Context, systemPrompt, sessionID, message string) (cliResult, error) {
	args := []string{"--print", "--output-format", "json", "--allowedTools", "Read,Glob,Grep,WebFetch,WebSearch"}
	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	} else if systemPrompt != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}
	args = append(args, message)

	out, err := exec.CommandContext(ctx, "claude", args...).Output()
	if err != nil {
		return cliResult{}, err
	}

	var resp struct {
		Result    string `json:"result"`
		SessionID string `json:"session_id"`
		IsError   bool   `json:"is_error"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return cliResult{}, fmt.Errorf("parse output: %w", err)
	}
	if resp.IsError {
		return cliResult{}, fmt.Errorf("%s", resp.Result)
	}
	return cliResult{text: resp.Result, sessionID: resp.SessionID}, nil
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
