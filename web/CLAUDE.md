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
| `/contribute` | How to submit a guild/solo |
| `/how-it-works` | How it works |

## Rules

- **Never hardcode internal paths in React components** — always use `url('/path')` from `src/lib/url.ts`
- Base path configured in `astro.config.mjs` via `ASTRO_BASE` env var (default: `/awesome-wwm-base-building`)
- Do NOT mix web code with Go code
- **Always use shadcn/ui — never write custom CSS or custom UI components from scratch.** Use shadcn primitives (`src/components/ui/`) for all UI needs. If a needed component is not yet installed, propose installing it via `npx shadcn@latest add <component>` rather than implementing it manually.
- **Proactively suggest shadcn components**: when reviewing or building UI, identify shadcn primitives that could replace hand-rolled markup or inline styles and recommend them.
- **Button-styled links in Astro pages**: use `buttonVariants` on `<a>` tags (`class={buttonVariants({ variant, size })}`), not `Button asChild` — Radix Slot requires React hydration
