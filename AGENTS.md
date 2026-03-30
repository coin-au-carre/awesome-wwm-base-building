# AGENTS.md - Code Guidelines for AI Coding Agents

This document provides comprehensive guidelines for AI agents working on the awesome-wwm-base-building codebase.

## Important: Primary Website Content

**README.md is the main content of the website and should be the first thing to understand when working on this project.** The README.md file:
- Serves as the homepage and primary content for the static site
- Contains the curated list of WWM guild bases
- Is automatically updated by the generator between HTML comment markers
- Should be read first to understand the project's purpose and content structure
- All changes to the guild listing happen through the generator, not manual edits

## Quick Reference

- **Language**: Go 1.24.0
- **Build System**: Task (https://taskfile.dev/)
- **Module Name**: `ruby`
- **Linter**: golangci-lint --fast
- **Project Type**: Discord bot + static site generator for WWM guild showcases
- **Main Website Content**: README.md (read this first!)

## Build, Lint, and Test Commands

### Core Development Commands
```bash
# Run go vet for code analysis
task vet

# Generate static pages from guilds.json
task generate
task generate -- -clean  # Remove stale guild pages

# Sync data from Discord (with git pull)
task sync
task sync:nopull        # Skip git pull
task sync -- -dry-run   # Preview changes without writing
task sync -- -no-notify # Skip Discord notification

# Run Discord bot
task bot               # Run in production mode (RUBY_CHANNEL_ID)
task bot -- -dev       # Run in dev mode (DEV_CHANNEL_ID)

# Full workflow
task all               # sync + generate
task all:push          # sync + generate + git push
```

### Testing
Currently, the project does not have unit tests. When adding tests:
- Create `*_test.go` files alongside the code being tested
- Use `go test ./...` to run all tests
- Test timeout is configured to 360s in VSCode settings

### Building and Deployment
```bash
task bot:build         # Build Linux binary → dist/bot
task bot:deploy        # Build + deploy to VPS
```

## Code Style Guidelines

### Package Structure
```
/cmd/           # Executable commands (thin main.go files)
/internal/      # Core business logic
  /discord/     # Discord integration
  /generator/   # Static site generation
  /guild/       # Domain models and parsing
/guilds/        # Generated markdown files (DO NOT EDIT)
/assets/        # Static assets
```

### Import Organization
Always organize imports in three groups with blank lines between:
```go
import (
    // 1. Standard library
    "context"
    "fmt"
    "log/slog"
    
    // 2. Internal packages (use module name "ruby")
    "ruby/internal/discord"
    "ruby/internal/guild"
    
    // 3. Third-party packages
    "github.com/bwmarrin/discordgo"
    "github.com/joho/godotenv"
)
```

### Naming Conventions

**Functions and Methods:**
- Public: `PascalCase` (e.g., `NewBot`, `ParseFirstPost`)
- Private: `camelCase` (e.g., `requireEnv`, `cleanSection`)
- Event handlers: `onEventName()` pattern returning closures

**Variables and Constants:**
- Variables: `camelCase` (e.g., `channelID`, `guildMap`)
- Constants: `PascalCase` or `SCREAMING_SNAKE_CASE`
- Group related constants in const blocks

**Types:**
- Structs: `PascalCase` (e.g., `Bot`, `Guild`, `SyncStats`)
- Use pointer receivers for methods that modify state
- Add JSON tags with `omitempty` for optional fields

### Error Handling

**In main functions:**
```go
if err != nil {
    slog.Error("operation failed", "err", err)
    os.Exit(1)
}
```

**In library code:**
```go
if err != nil {
    return nil, fmt.Errorf("creating session: %w", err)
}
```

**For non-critical errors:**
```go
if err != nil {
    slog.Warn("optional operation failed", "err", err)
    // Continue execution
}
```

### Logging
Use structured logging with `log/slog`:
```go
slog.Info("operation started", "channel", channelID, "user", username)
slog.Error("operation failed", "err", err, "id", messageID)
slog.Warn("degraded operation", "reason", "rate limited")
```

### Comments and Documentation

**Function documentation:**
```go
// ParseFirstPost extracts structured data from the first message of a Discord thread.
// It returns the parsed fields and any parsing errors encountered.
func ParseFirstPost(content string) (...) { ... }
```

**Implementation comments:**
```go
// Strip mention tags and trim for clean text
text := strings.TrimSpace(reMention.ReplaceAllString(m.Content, ""))

// Fast path: single keyword commands skip AI processing
if spotlightKeywords[strings.ToLower(text)] { ... }
```

### Type Definitions

**Struct with JSON tags:**
```go
type Guild struct {
    ID         string   `json:"id,omitempty"`
    Name       string   `json:"name"`
    GuildName  string   `json:"guildName,omitempty"`
    Tags       []string `json:"tags,omitempty"`
}
```

**Configuration pattern:**
```go
type Config struct {
    ReadmePath string
    GuildsDir  string
}

func DefaultConfig() Config {
    return Config{
        ReadmePath: "README.md",
        GuildsDir:  "guilds",
    }
}
```

### Concurrency Patterns

**Worker pool for parallel processing:**
```go
jobs := make(chan Job, len(items))
var wg sync.WaitGroup

// Start workers
for i := 0; i < numWorkers; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for job := range jobs {
            // Process job
        }
    }()
}

// Send jobs
for _, item := range items {
    jobs <- item
}
close(jobs)
wg.Wait()
```

### Regular Expressions
Compile at package level for reuse:
```go
var (
    reBracketID  = regexp.MustCompile(`[\[(](\d+)[\])]`)
    reEightDigit = regexp.MustCompile(`\b(\d{8})\b`)
)
```

## Environment Variables

Required environment variables (set in `.env` file):
- `RUBY_BOT_TOKEN` - Discord bot token
- `ANTHROPIC_API_KEY` - Claude API key
- `GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID` - Discord forum channel ID
- `RUBY_CHANNEL_ID` - Production bot channel
- `DEV_CHANNEL_ID` - Development bot channel

Optional environment variables:
- `BOT_CHANNEL_ID` - Bot notification channel
- `BASE_BUILDER_ROLE_ID` - Discord role for base builders

## Working with Generated Files

- Files in `/guilds/` directory are auto-generated - DO NOT EDIT
- Run `task generate -- -clean` to remove orphaned guild pages
- The generator injects content between HTML comment markers in README.md

## Discord Integration Notes

- Use `discordgo` for all Discord API interactions
- Always check rate limits and handle them gracefully
- Log Discord operations with channel/user context
- Implement event handlers as closures for clean separation

## Git Workflow

- Run `task vet` before committing (or use `task push`)
- Generated files should be committed (guilds/*.md, README.md updates)
- Use meaningful commit messages describing the "why"
- The CI/CD pipeline runs sync automatically via GitHub Actions

## Common Patterns

**Required environment variable helper:**
```go
func requireEnv(key string) string {
    v := os.Getenv(key)
    if v == "" {
        log.Fatalf("missing required env var: %s", key)
    }
    return v
}
```

**File path handling:**
```go
filepath.Join(*rootDir, "guilds", guild.Slug()+".md")
```

**String normalization:**
```go
strings.TrimSpace(strings.ToLower(input))
```

When working on this codebase, prioritize clarity, maintain consistency with existing patterns, and always handle errors appropriately.