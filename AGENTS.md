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
  sync/         # Crawl Discord → update data/guilds.json + data/solos.json
  bot/          # Long-running Discord bot with Claude AI
  send/         # One-shot message sender
  spotlight/    # Post random guild screenshot to Discord
  events-sync/  # Fetch Discord Scheduled Events → data/events.json
  announce/     # Test-post a new-guild announcement card
  genjson/      # Generate public JSON for the Astro site
/internal/
  discord/    # Sync logic, scoring, voter weights, roles, bot handlers
  guild/      # Guild struct, JSON store (LoadFile/SaveFile), parser
/web/         # Astro static site — DO NOT mix with Go code
data/guilds.json   # Authoritative guild data — source of truth
data/solos.json    # Authoritative solo build data — source of truth
data/events.json   # Discord Scheduled Events — written by cmd/events-sync
README.md     # Dev documentation
```

## Key Data Files

- `data/guilds.json` / `data/solos.json` — written by `cmd/sync`, read by the Astro site at build time
- `data/events.json` — written by `cmd/events-sync`, synced every 30 min via CI

## Task Commands

```bash
task sync              # git pull + crawl Discord → update data/guilds.json + data/solos.json
task sync:nopull       # crawl only (no git pull)
task sync -- -dry-run      # crawl without writing JSON
task sync -- -no-notify    # skip Discord notification
task sync -- -force-role   # reassign roles to all authors

task events-sync       # fetch Discord Scheduled Events → data/events.json

task web               # start Astro dev server (localhost:4321)
task web:build         # production Astro build → web/dist/
task web:preview       # preview production build

task bot               # run Discord bot (dev)
task bot:build         # build Linux binary → dist/bot
task bot:deploy        # build + scp + restart systemd on VPS

task send              # post a message as Ruby (one-shot)
task spotlight         # post a random guild screenshot to the Ruby channel
task announce          # test-post a new-guild announcement card

task test              # go test ./...
task vet               # go vet ./...
task push              # vet + test + git push
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

- `sync.yml` — runs `task sync` on schedule (8×/day), commits `data/guilds.json data/solos.json`, triggers deploy on completion
- `sync-events.yml` — runs `cmd/events-sync` every 30 min, commits `data/events.json` if changed, push to main triggers deploy
- `deploy.yml` — triggered by push to `main` or on completion of "Sync Guild Data" or "Sync Events", uses `withastro/action@v3` with `path: web`
- `test.yml` — runs `go test ./...` on every push/PR to `main`
- GitHub Pages source must be set to **GitHub Actions** (not "Deploy from branch")

## Analytics & Monitoring

### Tools in use

- **Umami** (cloud.umami.is) — primary event analytics. Script injected in `web/src/layouts/main.astro` (website ID `b935013b-6c75-412f-bae6-c35e5bd65858`). Do not remove or move this script.
- **Cloudflare Web Analytics** — passive traffic analytics. Same layout file. Do not remove.
- **Google Search Console** — verified via DNS. Sitemap submitted: `https://www.wherebuildersmeet.com/sitemap-index.xml` (declared in `web/public/robots.txt`).

### Conventions

- HTML elements: `data-umami-event="event_name"` + `data-umami-event-<prop>="value"` attributes.
- JS: `window.umami?.track("event_name", { key: value })` — always optional-chain (`?.`), never assume `umami` is defined.
- Event names: `snake_case`. Property keys: `snake_case`. Keep names consistent with the table above when extending existing events.
- Do not add a `page` property — Umami records the current URL automatically with every event.

### When to suggest analytics improvements

Proactively suggest adding tracking when:
- A new interactive element is added (button, filter, tab, modal) — add a `umami?.track` call.
- A new CTA links to Discord — add `data-umami-event="discord_cta_click"`.
- A new page is added — ensure any navigation CTAs on it fire events with a `page` property.
- A new outbound link is added — consider whether the destination is worth tracking.

## Git Workflow

- Run `task vet` and `task test` before pushing (`task push` does both automatically)
- Generated files (`data/guilds.json`, `data/solos.json`, `data/events.json`) are committed by CI
- Use `task sync -- -dry-run` to test crawling without side effects
