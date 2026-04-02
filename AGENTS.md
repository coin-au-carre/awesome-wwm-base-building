# AGENTS.md - Code Guidelines for AI Coding Agents

This document provides comprehensive guidelines for AI agents working on the awesome-wwm-base-building codebase.

## Architecture Overview

```
Discord forum ──► task sync ──► guilds.json / solos.json ──► task generate ──► SHOWCASE.md + guilds/*.md
                                         │
                                         └──► Astro build (web/) ──► GitHub Pages
```

**Stack:**
- **Go 1.24** — Discord sync bot, data parser, markdown generator
- **Astro 5 + shadcn/ui + Tailwind 4** — static website (`web/`)
- **GitHub Actions** — `sync.yml` (data) + `deploy.yml` (site)
- **Module name:** `ruby`

## Project Structure

```
/cmd/
  sync/       # Crawl Discord → update guilds.json + solos.json
  generate/   # JSON → SHOWCASE.md + SOLO_SHOWCASE.md + guilds/*.md + solos/*.md
  bot/        # Long-running Discord bot with Claude AI
  send/       # One-shot message sender
  spotlight/  # Post random guild screenshot to Discord
/internal/
  discord/    # Sync logic, scoring, voter weights, roles, bot handlers
  generator/  # Markdown page + table builder
  guild/      # Guild struct, JSON store (LoadFile/SaveFile), parser
/web/         # Astro static site — DO NOT mix with Go code
/guilds/      # Auto-generated guild markdown pages (DO NOT EDIT)
/solos/       # Auto-generated solo build markdown pages (DO NOT EDIT)
guilds.json   # Authoritative guild data — source of truth
solos.json    # Authoritative solo build data — source of truth
SHOWCASE.md   # Auto-generated guild ranking (DO NOT EDIT)
SOLO_SHOWCASE.md  # Auto-generated solo ranking (DO NOT EDIT)
README.md     # Dev documentation
```

## Key Data Files

- `guilds.json` / `solos.json` — written by `cmd/sync`, read by `cmd/generate` and the Astro site at build time
- `SHOWCASE.md` / `SOLO_SHOWCASE.md` — injected between HTML comment markers by `cmd/generate`; do not edit manually
- Files in `guilds/` and `solos/` are auto-generated — **DO NOT EDIT**

## Task Commands

```bash
task sync              # git pull + crawl Discord → update guilds.json + solos.json
task sync:nopull       # crawl only (no git pull)
task sync -- -dry-run      # crawl without writing JSON
task sync -- -no-notify    # skip Discord notification
task sync -- -force-role   # reassign roles to all authors

task generate          # JSON → markdown pages + inject SHOWCASE.md
task generate -- -clean    # also remove stale pages

task all               # sync + generate (used by CI)
task all:push          # sync + generate + git push

task web               # start Astro dev server (localhost:4321)
task web:build         # production Astro build → web/dist/
task web:preview       # preview production build

task bot               # run Discord bot (dev)
task bot:build         # build Linux binary → dist/bot
task bot:deploy        # build + scp + restart systemd on VPS

task vet               # go vet ./...
task push              # vet + git push
```

## Environment Variables

```
RUBY_BOT_TOKEN                        # required — Discord bot token
GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID  # required — guild forum channel ID
SOLO_BUILD_SHOWCASE_CHANNEL_FORUM_ID  # optional — solo forum channel ID
BOT_CHANNEL_ID                        # optional — bot notification channel
RUBY_CHANNEL_ID                       # production bot channel
DEV_CHANNEL_ID                        # dev bot channel
BASE_BUILDER_ROLE_ID                  # Discord role for builders
BASE_CRITIC_ROLE_ID                   # Discord role for voters
ANTHROPIC_API_KEY                     # Claude AI (bot feature)
```

## Scoring & Voter Weights

- `⭐` = 2 pts, `👍` / `🔥` = 1 pt each
- Lore bonus: +1, What to Visit bonus: +1
- Voter weight is computed from **combined** guild + solo thread reactions:
  - 2+ threads → ×1, 6+ → ×2, 12+ → ×3
- `discord.CollectVoterCounts(bot, channelID)` fetches counts for one channel
- `discord.MergeVoterCounts(a, b)` adds them together
- `discord.ComputeVoterWeights(counts)` converts to multipliers
- Pass merged weights via `SyncConfig.ExternalVoterWeights` to override internal computation

## Go Code Style

### Import Groups
```go
import (
    // 1. Standard library
    "fmt"
    "log/slog"

    // 2. Internal packages
    "ruby/internal/discord"
    "ruby/internal/guild"

    // 3. Third-party
    "github.com/bwmarrin/discordgo"
)
```

### Error Handling
```go
// main functions — fatal
slog.Error("operation failed", "err", err)
os.Exit(1)

// library code — wrap and return
return nil, fmt.Errorf("creating session: %w", err)

// non-critical — warn and continue
slog.Warn("optional operation failed", "err", err)
```

### Logging
```go
slog.Info("threads collected", "count", len(threads), "elapsed", time.Since(t0))
```

### Concurrency — worker pool
```go
jobs := make(chan Job, len(items))
var wg sync.WaitGroup
for range numWorkers {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for job := range jobs { /* process */ }
    }()
}
for _, item := range items { jobs <- item }
close(jobs)
wg.Wait()
```

### guild.LoadFile / SaveFile
Use `guild.LoadFile(path)` and `guild.SaveFile(path, guilds)` for arbitrary paths.
`guild.Load(root)` / `guild.Save(root, guilds)` are convenience wrappers for `guilds.json` only.

### generator.Config
```go
generator.Generate(guilds, generator.Config{
    ReadmePath:  "SHOWCASE.md",   // target markdown file
    GuildsDir:   "guilds",        // output directory
    PagesSubdir: "guilds",        // used in markdown links
    Clean:       true,            // remove stale pages
})
```

### Regular Expressions
Compile at package level:
```go
var reGuildName = regexp.MustCompile(`(?m)^[#\s]*(?::[^:]+:|\*\*|\p{So}\s*)*(.+?)\**\s*[\[(]\d{6,9}[\])]`)
```
- `\p{So}` matches Unicode symbol/emoji (e.g. 🏯)
- `\p{L}`, `\p{N}` used in `Slugify()` for Unicode-aware slug generation

## Astro Site (`web/`)

**Key files:**
- `src/lib/guilds.ts` — loads `guilds.json` and `solos.json` at build time via `readFileSync`
- `src/lib/slugify.ts` — port of Go `Slugify()` — **must stay in sync** with `internal/generator/page.go`
- `src/lib/url.ts` — `url(path)` helper for internal links inside React components (handles `BASE_URL`)
- `src/types/guild.ts` — TypeScript mirror of `internal/guild/guild.go`
- `src/components/GuildTable.tsx` — `client:load` React island, accepts `basePath` prop (`"guilds"` or `"solos"`)
- `src/components/TopShowcase.astro` — accepts `basePath` prop
- `src/components/MediaGallery.astro` — lightbox, YouTube embed, `onerror` fallback

**Pages:**
| Route | Description |
|---|---|
| `/` | Guild bases index (showcase + ranked table) |
| `/guilds/[slug]` | Guild detail page |
| `/solo` | Solo builds index |
| `/solos/[slug]` | Solo build detail page |
| `/items` | Building items catalog (WIP) |
| `/scoring` | Scoring rules |
| `/about` | About the project |
| `/contribute` | How to submit a guild/solo |

**Base path:** configured in `astro.config.mjs` via `ASTRO_BASE` env var (default: `/awesome-wwm-base-building`).

**Never hardcode internal paths in React components** — always use `url('/path')` from `src/lib/url.ts`.

## GitHub Actions

- `sync.yml` — runs `task all` on schedule (2×/day), commits `guilds.json solos.json SHOWCASE.md SOLO_SHOWCASE.md guilds/ solos/`, push triggers deploy
- `deploy.yml` — triggered by push to `main`, uses `withastro/action@v3` with `path: web`
- GitHub Pages source must be set to **GitHub Actions** (not "Deploy from branch")

## Git Workflow

- Run `task vet` before committing (`task push` does this automatically)
- Generated files (`guilds/*.md`, `solos/*.md`, `SHOWCASE.md`, `SOLO_SHOWCASE.md`) are committed by CI
- Use `task sync -- -dry-run` to test crawling without side effects
