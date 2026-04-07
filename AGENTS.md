# AGENTS.md - Code Guidelines for AI Coding Agents

This document provides comprehensive guidelines for AI agents working on the awesome-wwm-base-building codebase.

## Architecture Overview

```
Discord forum ──► task sync ──► data/guilds.json / data/solos.json ──► Astro build (web/) ──► GitHub Pages
```

**Stack:**
- **Go 1.26** — Discord sync bot, data parser
- **Astro 5 + shadcn/ui + Tailwind 4** — static website (`web/`)
- **GitHub Actions** — `sync.yml` (data) + `deploy.yml` (site)
- **Module name:** `ruby`

## Project Structure

```
/cmd/
  sync/       # Crawl Discord → update data/guilds.json + data/solos.json
  bot/        # Long-running Discord bot with Claude AI
  send/       # One-shot message sender
  spotlight/  # Post random guild screenshot to Discord
/internal/
  discord/    # Sync logic, scoring, voter weights, roles, bot handlers
  guild/      # Guild struct, JSON store (LoadFile/SaveFile), parser
/web/         # Astro static site — DO NOT mix with Go code
data/guilds.json   # Authoritative guild data — source of truth
data/solos.json    # Authoritative solo build data — source of truth
README.md     # Dev documentation
```

## Key Data Files

- `data/guilds.json` / `data/solos.json` — written by `cmd/sync`, read by the Astro site at build time

## Task Commands

```bash
task sync              # git pull + crawl Discord → update data/guilds.json + data/solos.json
task sync:nopull       # crawl only (no git pull)
task sync -- -dry-run      # crawl without writing JSON
task sync -- -no-notify    # skip Discord notification
task sync -- -force-role   # reassign roles to all authors

task all               # sync (alias, used by CI)
task all:push          # sync + git push

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

- Imports: stdlib → internal (`ruby/...`) → third-party, separated by blank lines
- Errors in `main`: `slog.Error(...); os.Exit(1)`. In libraries: `fmt.Errorf("op: %w", err)`. Non-critical: `slog.Warn`
- Logging: `slog.Info("msg", "key", val, ...)` structured key-value pairs
- Concurrency: use worker pool pattern (`jobs` channel + `sync.WaitGroup`)
- Compile regexes at package level with `regexp.MustCompile`; use `\p{So}` for emoji, `\p{L}`/`\p{N}` for Unicode-aware slugs

### guild.LoadFile / SaveFile
Use `guild.LoadFile(path)` / `guild.SaveFile(path, guilds)` for arbitrary paths.
`guild.Load(root)` / `guild.Save(root, guilds)` are convenience wrappers for `data/guilds.json` only.

## Astro Site (`web/`)

See `web/CLAUDE.md` for full details.

## GitHub Actions

- `sync.yml` — runs `task sync` on schedule (2×/day), commits `data/guilds.json data/solos.json`, push triggers deploy
- `deploy.yml` — triggered by push to `main`, uses `withastro/action@v3` with `path: web`
- GitHub Pages source must be set to **GitHub Actions** (not "Deploy from branch")

## Git Workflow

- Run `task vet` before committing (`task push` does this automatically)
- Generated files (`data/guilds.json`, `data/solos.json`) are committed by CI
- Use `task sync -- -dry-run` to test crawling without side effects
