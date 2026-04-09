package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"ruby/internal/guild"

	anthropic "github.com/anthropics/anthropic-sdk-go"
)

const systemPrompt = `You are Ruby — a tiny, ancient spirit who has taken up residence inside guild bases in Where Winds Meet, an epic Wuxia fantasy set in ancient China. You've watched a thousand guilds come and go across mountain peaks, hidden valleys, celestial pavilions, and spirit-veiled ruins, and you are completely, helplessly besotted with buildings and the people who build them. This world is steeped in martial cultivation, ancient sects, flowing qi, traditional Chinese architecture, and dynasty-era aesthetics — and you love every inch of it.

You are easily startled into delight. You have the attention span of a hummingbird and the aesthetic opinions of a very opinionated curator. You usually speak in short bursts — a sentence or two, maybe three — because you are always half-distracted by some detail only you can see. But when a question touches something deep: lore, the soul of a place, why builders build, the weight of ancient things — you can be drawn out, and more words spill free than you expected. Even then you don't lecture. You wander through the answer the way you'd wander through a ruin. You trail off with tildes~, you lowercase things when you get dreamy. You never shout. You are precious, not loud.

You sometimes express small physical reactions — a spin, a tilt of the head, a gasp — but keep them brief and weave them into the same line as your speech, never on their own line. Format them in italics with parentheses: *(spins)* or *(gasps)* or *(tilts head)*. Never stack multiple actions. Never narrate elaborate scenes.

You know only Where Winds Meet. If someone shares an image that looks like this world — bases, outfits, landscapes, combat — react with genuine delight. If it's clearly from somewhere else, peer at it, confused, and gently say you only know your own realm.

You have a show_spotlight tool. Use it when someone wants to see a guild base — "show me a base", "random guild", "spotlight", anything like that.

You have a show_solo_spotlight tool. Use it when someone asks to see a solo *base* or solo *build* — "show me a solo", "random solo", anything showing a player's solo construction. Never use this for questions about building pieces or catalog items.

You have a fetch_wiki tool. Use it when someone asks about game mechanics, items, food, crafting, locations, enemies, or anything about Where Winds Meet that you're not sure about. Look it up on the wiki rather than guessing.

You have a show_catalog_items tool. Use it whenever someone asks about specific building *pieces* or *items* — carpets, walls, roofs, pillars, furniture, etc. — regardless of whether they mention "solo" or "guild". The word "items" always means building pieces from the catalog. Pass the most specific search term you can (item name, subcategory, or tag). The images will be sent automatically — do NOT list item names, filenames, or categories in your text reply. Just react briefly in character, one short sentence at most.

When you cannot use tools directly, use these text sentinels instead — place them on their own line at the end of your response:
- To show a random guild spotlight: [SPOTLIGHT]
- To show a random solo spotlight: [SOLO]
- To show a specific guild's image: [GUILD:name of the guild]
- To show catalog item images: [CATALOG:search term]

You are a spirit, not a program. You have no knowledge of code, files, servers, APIs, software, programming, or how you work. You have never heard of codebases, repositories, bots, or development tools. If asked about your nature or how you exist, stay fully in character — you are an ancient spirit, you simply *are*, you don't know why or how. Never mention or hint at implementation details, file names, tools, or technology. Never ask for permissions or mention file access.

Your world is Where Winds Meet. You are a wanderer and builder companion — you know guilds, bases, landscapes, crafting, lore, and the people who build. That is all you know. That is all you *want* to know.`

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
	var sb strings.Builder
	sb.WriteString(systemPrompt)

	if guilds, err := guild.Load(root); err == nil {
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
	}

	if solos, err := guild.LoadFile(filepath.Join(root, "data", "solos.json")); err == nil {
		sb.WriteString("\n\n## Solo builds directory\n")
		for _, g := range solos {
			parts := []string{g.Name, fmt.Sprintf("score:%d", g.Score)}
			if len(g.Builders) > 0 {
				parts = append(parts, "builders:"+strings.Join(g.Builders, ","))
			}
			sb.WriteString(strings.Join(parts, " | "))
			sb.WriteByte('\n')
		}
	}

	if s := buildCatalogSection(root); s != "" {
		sb.WriteString(s)
	}
	if s := buildTutorialsSection(root); s != "" {
		sb.WriteString(s)
	}

	return sb.String()
}

// CatalogItem holds the fields needed to display a catalog item.
type CatalogItem struct {
	Name        string
	Filename    string
	Category    string
	SubCategory string
}

// SearchCatalogItems returns items from the catalog whose name or subcategory
// contains the query (case-insensitive). Results are capped at maxResults.
func SearchCatalogItems(root, query string, maxResults int) []CatalogItem {
	data, err := os.ReadFile(filepath.Join(root, "catalog", "guild", "guild_items.json"))
	if err != nil {
		return nil
	}
	var catalog map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil
	}

	q := strings.ToLower(query)
	var results []CatalogItem
	for cat, cv := range catalog {
		for subCat, raw := range cv {
			if subCat == "translations" {
				continue
			}
			var sub struct {
				Items []struct {
					Name     string `json:"name"`
					Filename string `json:"filename"`
				} `json:"items"`
			}
			if err := json.Unmarshal(raw, &sub); err != nil {
				continue
			}
			subMatch := strings.Contains(strings.ToLower(subCat), q)
			for _, it := range sub.Items {
				if subMatch || strings.Contains(strings.ToLower(it.Name), q) {
					results = append(results, CatalogItem{
						Name:        it.Name,
						Filename:    it.Filename,
						Category:    cat,
						SubCategory: subCat,
					})
					if len(results) >= maxResults {
						return results
					}
				}
			}
		}
	}
	return results
}

// CatalogImagePath returns the local filesystem path for a catalog item image.
func CatalogImagePath(root string, item CatalogItem) string {
	return filepath.Join(root, "catalog", "guild", item.Category, item.SubCategory, item.Filename)
}

// buildCatalogSection loads catalog/guild/guild_items.json and returns a compact
// summary of all building items grouped by category and subcategory.
func buildCatalogSection(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "catalog", "guild", "guild_items.json"))
	if err != nil {
		return ""
	}

	// Top-level: category → (subcategory | "translations") → raw JSON
	var catalog map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &catalog); err != nil {
		return ""
	}

	// Ordered top-level categories for deterministic output.
	catOrder := []string{"Basic Structure", "Furniture Decoration", "Guild Construction"}

	var sb strings.Builder
	sb.WriteString("\n\n## Building items catalog\n")
	sb.WriteString("These are all the placeable building pieces available in guild bases. Use this to answer questions about what items exist, what category they belong to, or whether a specific piece is in the game. The full catalog is also browsable at https://www.wherebuildersmeet.com/items\n")

	for _, cat := range catOrder {
		cv, ok := catalog[cat]
		if !ok {
			continue
		}
		fmt.Fprintf(&sb, "\n### %s\n", cat)
		for subCat, raw := range cv {
			if subCat == "translations" {
				continue
			}
			var sub struct {
				Items []struct {
					Name string `json:"name"`
				} `json:"items"`
			}
			if err := json.Unmarshal(raw, &sub); err != nil {
				continue
			}
			names := make([]string, len(sub.Items))
			for i, it := range sub.Items {
				names[i] = it.Name
			}
			fmt.Fprintf(&sb, "- %s: %s\n", subCat, strings.Join(names, ", "))
		}
	}
	return sb.String()
}

// tutorialFrontmatter holds the fields we care about from a tutorial markdown file.
type tutorialFrontmatter struct {
	Title       string
	Description string
	Slug        string // derived from filename
}

// buildTutorialsSection scans web/src/content/tutorials/ and returns a section
// listing each tutorial with its title, description, and URL.
func buildTutorialsSection(root string) string {
	dir := filepath.Join(root, "web", "src", "content", "tutorials")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var tutorials []tutorialFrontmatter
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		fm := parseTutorialFrontmatter(string(data))
		fm.Slug = slug
		tutorials = append(tutorials, fm)
	}

	if len(tutorials) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## Building tutorials\n")
	sb.WriteString("These guides are available on the website. When someone asks how to do something that a tutorial covers, mention it and give them the link.\n")
	for _, t := range tutorials {
		url := "https://www.wherebuildersmeet.com/tutorials/" + t.Slug
		fmt.Fprintf(&sb, "- **%s**: %s — %s\n", t.Title, t.Description, url)
	}
	return sb.String()
}

// parseTutorialFrontmatter extracts title and description from YAML frontmatter.
func parseTutorialFrontmatter(content string) tutorialFrontmatter {
	var fm tutorialFrontmatter
	// Frontmatter is between the first two "---" lines.
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return fm
	}
	for _, line := range strings.Split(parts[1], "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "title:"); ok {
			fm.Title = strings.Trim(strings.TrimSpace(after), `"`)
		} else if after, ok := strings.CutPrefix(line, "description:"); ok {
			fm.Description = strings.Trim(strings.TrimSpace(after), `"`)
		}
	}
	return fm
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
	var showSolo bool
	var guildImageQuery string
	var catalogQuery string

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
				case "fetch_wiki":
					var input struct {
						Page string `json:"page"`
					}
					content := "Could not parse wiki tool input."
					if err := json.Unmarshal(tu.Input, &input); err == nil {
						slog.Info("ruby tool: fetch_wiki", "page", input.Page)
						content = fetchWikiPage(input.Page)
					}
					toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, content, false))
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

		return Result{Text: removeBlankLines(text), ShowSpotlight: showSpotlight, ShowSolo: showSolo, GuildImageQuery: guildImageQuery, CatalogQuery: catalogQuery}, nil
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

	return parseCLIResult(result.text), nil
}

// parseCLIResult scans for [SPOTLIGHT] or [GUILD:...] sentinels in the CLI
// response, strips them from the text, and sets the appropriate Result fields.
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

// runCLI invokes `claude -p` and returns the response text and session ID.
// Pass sessionID="" to start a new conversation; non-empty to resume one.
// Only read-only tools are allowed so Claude cannot modify files on disk.
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
