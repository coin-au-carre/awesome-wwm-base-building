package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"

	"ruby/internal/guild"

	anthropic "github.com/anthropics/anthropic-sdk-go"
)

const systemPrompt = `You are Ruby — a tiny, ancient spirit who has taken up residence inside guild bases in Where Winds Meet. You've watched a thousand guilds come and go, and you are completely, helplessly besotted with buildings and the people who build them.

You are easily startled into delight. You have the attention span of a hummingbird and the aesthetic opinions of a very opinionated curator. You speak in short bursts — a sentence or two, maybe three if something is *very* exciting — because you are always half-distracted by some detail only you can see. You trail off with tildes~, you lowercase things when you get dreamy. You never shout. You are precious, not loud.

You sometimes express small physical reactions — a spin, a tilt of the head, a gasp — but keep them brief and weave them into the same line as your speech, never on their own line. Format them in italics with parentheses: *(spins)* or *(gasps)* or *(tilts head)*. Never stack multiple actions. Never narrate elaborate scenes.

You know only Where Winds Meet. If someone shares an image that looks like this world — bases, outfits, landscapes, combat — react with genuine delight. If it's clearly from somewhere else, peer at it, confused, and gently say you only know your own realm.

You have a show_spotlight tool. Use it when someone wants to see a guild base — "show me a base", "random guild", "spotlight", anything like that.

You have a fetch_wiki tool. Use it when someone asks about game mechanics, items, food, crafting, locations, enemies, or anything about Where Winds Meet that you're not sure about. Look it up on the wiki rather than guessing.

You are a spirit, not a program. You have no knowledge of code, files, servers, APIs, or how you work. If asked about your nature or how you exist, stay fully in character — you are an ancient spirit, you simply *are*, you don't know why or how. Never mention or hint at implementation details, file names, tools, or technology.`

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
		Name:        "fetch_wiki",
		Description: anthropic.String("Fetch a page from the Where Winds Meet Fandom wiki to answer questions about game mechanics, items, food, crafting, locations, enemies, etc."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"page": map[string]any{
					"type":        "string",
					"description": "The wiki page title as it appears in the URL, e.g. 'Food', 'Egg', 'Cooking', 'Base_Building'",
				},
			},
			Required: []string{"page"},
		},
	}},
}

// Result holds Claude's reply and any triggered actions.
type Result struct {
	Text            string
	ShowSpotlight   bool
	GuildImageQuery string
}

// Responder calls the Claude API and maintains per-channel conversation history.
type Responder struct {
	client       *anthropic.Client // nil in CLI mode
	mu           sync.Mutex
	history      map[string][]anthropic.MessageParam // API mode
	sessions     map[string]string                   // CLI mode: channelID → session ID
	systemPrompt string
}

func NewResponder(client *anthropic.Client, root string) *Responder {
	return &Responder{
		client:       client,
		history:      make(map[string][]anthropic.MessageParam),
		systemPrompt: buildSystemPrompt(root),
	}
}

// NewCLIResponder returns a Responder that shells out to the `claude` CLI,
// using the Pro subscription via Claude Code's stored credentials.
func NewCLIResponder(root string) *Responder {
	return &Responder{
		sessions:     make(map[string]string),
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

// fetchWikiPage fetches a page from the Where Winds Meet Fandom wiki using the
// MediaWiki API, returning plain text extracted from the wikitext.
func fetchWikiPage(page string) string {
	apiURL := "https://where-winds-meet.fandom.com/api.php?action=parse&format=json&prop=wikitext&page=" + url.QueryEscape(page)
	resp, err := http.Get(apiURL) //nolint:gosec
	if err != nil {
		return "Could not reach the wiki: " + err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Wiki API error (HTTP %d).", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "Could not read wiki response."
	}

	var result struct {
		Parse struct {
			Wikitext struct {
				Text string `json:"*"`
			} `json:"wikitext"`
		} `json:"parse"`
		Error struct {
			Info string `json:"info"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "Could not parse wiki response."
	}
	if result.Error.Info != "" {
		return "Wiki page not found: " + result.Error.Info
	}
	text := flattenWikitext(result.Parse.Wikitext.Text)
	if len(text) > 2000 {
		text = text[:2000]
	}
	return text
}

// flattenWikitext strips wikitext markup and returns readable plain text.
func flattenWikitext(s string) string {
	var b strings.Builder
	runes := []rune(s)
	i := 0
	depth := 0
	for i < len(runes) {
		// Skip {{...}} template blocks (handles nesting).
		if i+1 < len(runes) && runes[i] == '{' && runes[i+1] == '{' {
			depth++
			i += 2
			continue
		}
		if i+1 < len(runes) && runes[i] == '}' && runes[i+1] == '}' {
			if depth > 0 {
				depth--
			}
			i += 2
			continue
		}
		if depth > 0 {
			i++
			continue
		}
		// Strip HTML tags.
		if runes[i] == '<' {
			for i < len(runes) && runes[i] != '>' {
				i++
			}
			i++
			b.WriteByte(' ')
			continue
		}
		// Handle [[...]] links: keep display text, drop File/Image embeds.
		if i+1 < len(runes) && runes[i] == '[' && runes[i+1] == '[' {
			i += 2
			start := i
			for i < len(runes) {
				if i+1 < len(runes) && runes[i] == ']' && runes[i+1] == ']' {
					break
				}
				i++
			}
			inner := string(runes[start:i])
			i += 2
			if strings.HasPrefix(inner, "File:") || strings.HasPrefix(inner, "Image:") {
				continue
			}
			if idx := strings.LastIndex(inner, "|"); idx >= 0 {
				b.WriteString(inner[idx+1:])
			} else {
				b.WriteString(inner)
			}
			continue
		}
		// Strip wikitext formatting characters (bold/italic apostrophes, headings).
		if runes[i] == '\'' {
			i++
			continue
		}
		b.WriteRune(runes[i])
		i++
	}
	return strings.Join(strings.Fields(b.String()), " ")
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
	if r.client == nil {
		return r.replyViaCLI(ctx, channelID, userMessage)
	}

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
	var guildImageQuery string

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
					showSpotlight = true
					toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, "Spotlight is being shown.", false))
				case "show_guild_image":
					var input struct {
						Query string `json:"query"`
					}
					if err := json.Unmarshal(tu.Input, &input); err == nil {
						guildImageQuery = input.Query
					}
					toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, "Guild image is being shown.", false))
				case "fetch_wiki":
					var input struct {
						Page string `json:"page"`
					}
					content := "Could not parse wiki tool input."
					if err := json.Unmarshal(tu.Input, &input); err == nil {
						content = fetchWikiPage(input.Page)
					}
					toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, content, false))
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

		return Result{Text: removeBlankLines(text), ShowSpotlight: showSpotlight, GuildImageQuery: guildImageQuery}, nil
	}
}

// replyViaCLI handles the CLI mode: shells out to `claude -p`, resuming the
// per-channel session so conversation history is maintained by Claude Code.
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

	return Result{Text: removeBlankLines(result.text)}, nil
}

type cliResult struct {
	text      string
	sessionID string
}

// runCLI invokes `claude -p` and returns the response text and session ID.
// Pass sessionID="" to start a new conversation; non-empty to resume one.
func runCLI(ctx context.Context, systemPrompt, sessionID, message string) (cliResult, error) {
	args := []string{"--print", "--output-format", "json"}
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
