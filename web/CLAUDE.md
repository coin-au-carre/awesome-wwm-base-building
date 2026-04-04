# Astro Site (`web/`)

## Key Files

- `src/lib/guilds.ts` — loads `guilds.json` and `solos.json` at build time via `readFileSync`
- `src/lib/slugify.ts` — port of Go `Slugify()` — **must stay in sync with `internal/discord/spotlight.go`**
- `src/lib/url.ts` — `url(path)` helper for internal links inside React components (handles `BASE_URL`)
- `src/types/guild.ts` — TypeScript mirror of `internal/guild/guild.go`
- `src/components/GuildTable.tsx` — `client:load` React island, accepts `basePath` prop (`"guilds"` or `"solos"`)
- `src/components/TopShowcase.astro` — accepts `basePath` prop
- `src/components/MediaGallery.astro` — lightbox, YouTube embed, `onerror` fallback

## Pages

| Route | Description |
|---|---|
| `/` | Guild bases index (showcase + ranked table) |
| `/guilds/[slug]` | Guild detail page |
| `/solo` | Solo construction index |
| `/solos/[slug]` | Solo build detail page |
| `/items` | Building items catalog (WIP) |
| `/scoring` | Scoring rules |
| `/about` | About the project |
| `/contribute` | How to submit a guild/solo |

## Rules

- **Never hardcode internal paths in React components** — always use `url('/path')` from `src/lib/url.ts`
- Base path configured in `astro.config.mjs` via `ASTRO_BASE` env var (default: `/awesome-wwm-base-building`)
- Do NOT mix web code with Go code
