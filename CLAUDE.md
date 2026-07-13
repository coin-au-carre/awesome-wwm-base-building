# CLAUDE.md

> Keep this file updated when adding/removing files, changing scoring constants, or modifying data shapes.

See [web/CLAUDE.md](web/CLAUDE.md) for Astro site specifics.

## Quick reference

- Go module: `ruby`
- Run tasks via `task <name>`
- Vet + test before pushing: `task push` (runs `task vet` and `task test` automatically)
- Large data files (`data/guilds.json`, `data/solos.json`, `data/events.json`, `catalog/guild/guild_items.json`) are generated/managed by tooling — do not read in full unless explicitly needed
- `web/node_modules/` — ignore entirely

## Data shapes (do not read the full JSON files)

`data/guilds.json` / `data/solos.json` — array of:
```ts
interface Guild {
  id?: string
  name: string
  guildName?: string
  builders: string[]
  tags?: string[]
  discordThread: string
  posterDiscordId?: string
  posterUsername?: string
  postedOnBehalfOf?: string  // set when a mod posted on behalf of the builder
  lore?: string
  whatToVisit?: string
  score: number
  coverImage?: string
  screenshots?: string[]
  screenshotSections?: { label?: string; screenshots: string[] }[]
  videos?: string[]
  formerNames?: string[]   // past names, appended each time the thread is renamed in Discord
  createdAt?: string
  lastModified?: string
  scoutedByDiscordId?: string
}
```

`catalog/guild/guild_items.json` — nested object:
```
{ [category]: { translations, [subCategory]: { translations, items: [ { name, translations, filename, tags, styles, hasVariants, category, subCategory } ] } } }
```

## Architecture

```
Discord forum ──► task sync ──► data/guilds.json / data/solos.json ──► Astro build (web/) ──► GitHub Pages
```

- **Go 1.26** — Discord sync bot, data parser
- **Astro 6 + shadcn/ui + Tailwind 4** — static website (`web/`)
- **GitHub Actions** — `sync.yml` (data) + `deploy.yml` (site)

## Content Sources

| Content type | Source of truth | Sync command | Output |
|---|---|---|---|
| Guild/solo builds | Discord forum (crawled) | `task sync` | `data/guilds.json`, `data/solos.json` |
| Tutorials | Discord threads in `data/tutorial_threads.txt` | `task tutorial` | `web/src/content/articles/*.md` |
| Bugs | Google Sheets (CSV export) | `task sync:bugs` | `data/bugs.json` |
| Patch notes | Google Sheets (CSV export) | `task sync:patches` | `data/patches.json` |
| Events | Discord Scheduled Events API | `task events-sync` | `data/events.json` |
| Homestead level rankings | Discord guild member roles | `task homestead-sync` | `data/homestead_members.json` |

Tutorials are fetched from Discord threads listed one-per-line in `data/tutorial_threads.txt` and written as Markdown to `web/src/content/articles/`. Bugs and patch notes are managed in a shared Google Spreadsheet and exported as CSV at sync time.

### Sync thread-matching rules (`internal/discord/sync.go SyncFetch`)

| Situation | Result |
|---|---|
| Same name + same `discordThread` URL | Skip (already known) |
| Same name + different `discordThread` URL | Conflict warning to #dev, skip |
| Different name + same `discordThread` URL | **Rename**: update name in place, preserve `createdAt`, notify #dev |
| Same name + stored entry has no `discordThread` (placeholder) | **New guild**: append fresh entry, delete placeholder, set `createdAt` from thread |
| No existing entry | **New guild**: same as above |

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
  discord/    # Bot handlers split by feature: interactions.go (dispatcher),
              # submit/link/votes/builder.go (commands), sync.go (data pipeline),
              # llm/prompt/catalog.go (AI), score/roles/spotlight/util.go (support)
              # moderators.go — single source of truth for mod Discord IDs
  guild/      # Guild struct, JSON store (LoadFile/SaveFile), parser
/web/         # Astro static site — DO NOT mix with Go code
data/guilds.json   # Authoritative guild data — source of truth
data/solos.json    # Authoritative solo build data — source of truth
data/events.json   # Discord Scheduled Events — written by cmd/events-sync
```

## Task Commands

```bash
task sync              # git pull + crawl Discord → update data/guilds.json + data/solos.json
task sync:nopull       # crawl only (no git pull)
task sync -- -dry-run      # crawl without writing JSON
task sync -- -no-notify    # skip Discord notification
task sync -- -force-role   # reassign roles to all authors
task sync:guilds -- -guild "Name" # sync only one guild (skips tutorial/bugs/patches/whatever syncs)

task events-sync       # fetch Discord Scheduled Events → data/events.json

task web               # start Astro dev server (localhost:4321)
task web:build         # production Astro build → web/dist/
task web:preview       # preview production build

task bot               # run Discord bot (dev)

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
BLUEPRINT_CHANNEL_FORUM_ID           # optional — blueprint forum channel ID
BOT_CHANNEL_ID                        # optional — bot notification channel
RUBY_CHANNEL_ID                       # production bot channel
DEV_CHANNEL_ID                        # dev bot channel
BASE_BUILDER_ROLE_ID                  # Discord role for guild builders
SOLO_BUILDER_ROLE_ID                  # Discord role for solo builders (falls back to BASE_BUILDER_ROLE_ID if unset)
BASE_CRITIC_ROLE_ID                   # Discord role for voters
ANTHROPIC_API_KEY                     # Claude AI (bot feature)
BUGS_CHANNEL_ID                       # #bugs channel ID for the digest message
BUGS_MESSAGE_ID                       # pinned message ID to edit (empty = create new on first run, then save the logged ID)
UPDATES_CHANNEL_ID                    # #updates channel ID for the digest message
UPDATES_MESSAGE_ID                    # pinned message ID to edit (empty = create new on first run, then save the logged ID)
HOMESTEAD_MESSAGE_ID                   # pinned message ID in #homestead-hall-of-fame to edit (empty = create new on first run, then save the logged ID)
INSTANCE_LOCK_MESSAGE_ID               # message ID (in DEV_CHANNEL_ID, private) used as a cross-machine lock heartbeat (empty = falls back to discord.DefaultInstanceLockMessageID so both machines share one message even before either .env is updated)
INSTANCE_NAME                          # optional — label shown as the lock holder (defaults to os.Hostname())
INSTANCE_PRIORITY                      # optional int, default 0 — higher wins immediately over a lower-priority holder even if its heartbeat is fresh (set higher on local so it always preempts the VPS)
```

## Single-instance lock (local vs VPS)

`cmd/bot` can run on a local machine or the VPS, but never both at once — they share a Discord bot token, not a filesystem. The VPS instance is meant to run 24/7 as the fallback, so losing the lock must idle it, not kill the process. `discord.RunLocked` (`internal/discord/instancelock.go`) enforces this by cycling forever between "waiting" and "active", using a single Discord message in `DEV_CHANNEL_ID` (private, not the public bot channel) as a heartbeat lock, edited every 5s:
- While waiting, it reads the lock message. If the heartbeat is fresh (<15s old, i.e. no more than 3 missed beats), held by another instance, and that instance's priority is `>=` its own, it keeps polling — no gateway connection, no handlers firing, no data watcher.
- A higher-`INSTANCE_PRIORITY` instance (e.g. local) skips the wait and claims the lock immediately, even over a fresh lower-priority holder (e.g. VPS).
- Once it claims the lock, `onAcquire` runs (`bot.Open()` + `PullOnStart` + `StartDataWatcher`) and it starts heartbeating.
- Every heartbeat tick it re-reads the lock message; if a different holder now owns it (preempted by a higher-priority instance), it runs `onRelease` (`bot.Close()`), stops its data watcher, and goes back to waiting — the process itself keeps running so it can reclaim the lock automatically once the other instance goes away (e.g. a laptop going to sleep).
- Only a real shutdown (SIGINT/SIGTERM, i.e. the outer `ctx`) makes `RunLocked` return and the process exit.
- Killing the active bot lets the other, already-waiting instance take over automatically within ~15s — no manual restart needed.

> **When adding a new env var that affects `task sync`:** you must do all three or Discord CDN URLs in the output file will expire and break images on the live site:
> 1. Add to `.env` locally
> 2. Add as a secret in GitHub repo Settings → Secrets and variables → Actions
> 3. Add to `.github/workflows/sync.yml` — both in the `env:` block of the "Run sync" step, and in the `git add` line of the "Commit and push" step (for any new output data file)

## Scoring & Voter Weights

- `⭐` = 2 pts, `👍` / `🔥` = 1 pt each
- Lore bonus: +1, What to Visit bonus: +1
- Voter weight is computed from **combined** guild + solo thread reactions:
  - 4+ threads → ×1, 8+ → ×2, 12+ → ×3 (Critic)
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

### cmdutil helpers
- `cmdutil.LoadEnv(root)` — loads `<root>/.env`, warns and continues if absent. Call this at the top of every command's `main` after parsing flags.
- `cmdutil.RequireEnv(key)` — returns the env var value or exits with an error.
- `cmdutil.RootDir()` — returns `"."` when run from the repo root, `".."` otherwise.

## GitHub Actions

- `sync.yml` — runs `task sync` on schedule (8×/day), commits `data/guilds.json data/solos.json`, triggers deploy on completion
- `sync-events.yml` — runs `cmd/events-sync` every 30 min, commits `data/events.json` if changed, push to main triggers deploy
- `sync-homestead.yml` — runs `cmd/homestead-sync` hourly, commits `data/homestead_members.json`/`data/users.json` if changed (does not trigger deploy)
- `deploy.yml` — triggered by push to `main` or on completion of "Sync Guild Data" or "Sync Events", uses `withastro/action@v3` with `path: web`
- `test.yml` — runs `go test ./...` on every push/PR to `main`
- GitHub Pages source must be set to **GitHub Actions** (not "Deploy from branch")
