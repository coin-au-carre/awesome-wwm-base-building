package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
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
	AllowEmbeds     bool
}

// Responder calls the Claude API and maintains per-channel conversation history.
type Responder struct {
	client       *anthropic.Client
	mu           sync.Mutex
	history      map[string][]anthropic.MessageParam // API mode
	sessions     map[string]string                   // CLI mode: channelID → session ID
	systemPrompt string
	guilds       []promptGuild
	solos        []promptGuild
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
	{OfTool: &anthropic.ToolParam{
		Name:        "search_guilds",
		Description: anthropic.String("Search guild bases and solo builds by keyword. Use when asked which guilds or solo builds contain a specific element, theme, landmark, feature, creature, or physical object — including modern/unusual things like tanks, vehicles, computers, planes. (e.g. 'guilds with a tank', 'guilds with a dragon', 'solo builds with a castle', 'builds with water'). Returns an exact list from the data — always prefer this over recalling from memory."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"keyword": map[string]any{
					"type":        "string",
					"description": "Word or phrase to search for across guild names, tags, lore, and what-to-visit descriptions",
				},
			},
			Required: []string{"keyword"},
		},
	}},
}

func NewResponder(client *anthropic.Client, root string) *Responder {
	guilds := loadPromptGuilds(root)
	solos := loadPromptSolos(root)
	return &Responder{
		client:       client,
		history:      make(map[string][]anthropic.MessageParam),
		systemPrompt: buildSystemPrompt(root, guilds),
		guilds:       guilds,
		solos:        solos,
	}
}

func NewCLIResponder(root string) *Responder {
	guilds := loadPromptGuilds(root)
	solos := loadPromptSolos(root)
	return &Responder{
		sessions:     make(map[string]string),
		systemPrompt: buildSystemPrompt(root, guilds),
		guilds:       guilds,
		solos:        solos,
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

	var showSpotlight, showSolo, allowEmbeds bool
	var guildImageQuery, catalogQuery string

	for {
		resp, err := r.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeHaiku4_5,
			MaxTokens: 2048,
			System:    []anthropic.TextBlockParam{{Text: r.systemPrompt, CacheControl: anthropic.NewCacheControlEphemeralParam()}},
			Messages:  msgs,
			Tools:     tools,
		})
		if err != nil {
			return Result{}, fmt.Errorf("claude API: %w", err)
		}

		u := resp.Usage
		if u.CacheReadInputTokens > 0 {
			slog.Info("ruby cache hit", "read", u.CacheReadInputTokens, "input", u.InputTokens, "output", u.OutputTokens)
		} else {
			slog.Info("ruby cache miss", "created", u.CacheCreationInputTokens, "input", u.InputTokens, "output", u.OutputTokens)
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
				case "search_guilds":
					var input struct {
						Keyword string `json:"keyword"`
					}
					if err := json.Unmarshal(tu.Input, &input); err == nil {
						msg, count := searchBuilds(r.guilds, r.solos, input.Keyword)
						if count < 4 {
							allowEmbeds = true
						}
						slog.Info("ruby tool: search_guilds", "keyword", input.Keyword, "count", count)
						toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, msg, false))
					}
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

		return Result{Text: removeBlankLines(text), ShowSpotlight: showSpotlight, ShowSolo: showSolo, GuildImageQuery: guildImageQuery, CatalogQuery: catalogQuery, AllowEmbeds: allowEmbeds}, nil
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

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = os.TempDir() // run outside the project so CLAUDE.md is not loaded
	out, err := cmd.Output()
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

func searchBuilds(guilds []promptGuild, solos []promptGuild, keyword string) (string, int) {
	kw := strings.ToLower(keyword)
	var results []string
	for _, g := range guilds {
		if snippets := matchBuild(g, kw); len(snippets) > 0 {
			results = append(results, fmt.Sprintf("- %s (score:%d) — %s", g.Name, g.Score, strings.Join(snippets, "; ")))
		}
	}
	for _, s := range solos {
		if snippets := matchBuild(s, kw); len(snippets) > 0 {
			results = append(results, fmt.Sprintf("- [Solo] %s (score:%d) — %s", s.Name, s.Score, strings.Join(snippets, "; ")))
		}
	}
	if len(results) == 0 {
		return fmt.Sprintf("No guilds or solo builds found matching %q.", keyword), 0
	}
	if len(results) > 15 {
		return fmt.Sprintf("Too many builds match %q (%d results) — do not list them. Instead, tell the user in character that there are too many to name and ask them to be more specific.", keyword, len(results)), len(results)
	}
	note := "Mention ALL results listed above — do not skip any. Solo builds are prefixed with [Solo]."
	return fmt.Sprintf("%d builds found matching %q:\n%s\n\n%s", len(results), keyword, strings.Join(results, "\n"), note), len(results)
}

func matchBuild(g promptGuild, kw string) []string {
	var snippets []string
	if strings.Contains(strings.ToLower(g.Name), kw) {
		snippets = append(snippets, fmt.Sprintf("name: %q", g.Name))
	}
	for _, tag := range g.Tags {
		if strings.Contains(strings.ToLower(tag), kw) {
			snippets = append(snippets, fmt.Sprintf("tag: %q", tag))
			break
		}
	}
	if strings.Contains(strings.ToLower(g.Lore), kw) {
		snippets = append(snippets, fmt.Sprintf("lore: %q", matchSnippet(g.Lore, kw, 60)))
	}
	if strings.Contains(strings.ToLower(g.WhatToVisit), kw) {
		snippets = append(snippets, fmt.Sprintf("visit: %q", matchSnippet(g.WhatToVisit, kw, 60)))
	}
	return snippets
}

// matchSnippet returns up to contextLen chars around the first occurrence of kw in text.
func matchSnippet(text, kw string, contextLen int) string {
	idx := strings.Index(strings.ToLower(text), kw)
	if idx < 0 {
		return ""
	}
	start := max(0, idx-contextLen/2)
	end := min(len(text), idx+len(kw)+contextLen/2)
	snippet := text[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}
	return strings.ReplaceAll(snippet, "\n", " ")
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
