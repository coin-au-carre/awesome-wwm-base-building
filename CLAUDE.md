# CLAUDE.md

See [AGENTS.md](AGENTS.md) for full architecture, project structure, task commands, Go style, and GitHub Actions details.

See [web/CLAUDE.md](web/CLAUDE.md) for Astro site specifics.

## Quick reference

- Go module: `ruby`
- Run tasks via `task <name>` (see AGENTS.md for full list)
- Vet before committing: `task vet`
- Large data files (`guilds.json`, `solos.json`, `catalog/guild/guild_items.json`) are generated/managed by tooling — do not read in full unless explicitly needed
- `web/node_modules/` — ignore entirely

## Data shapes (do not read the full JSON files)

`guilds.json` / `solos.json` — array of:
```ts
interface Guild {
  id?: string
  name: string
  guildName?: string
  builders: string[]
  tags?: string[]
  discordThread: string
  builderDiscordId?: string
  lore?: string
  whatToVisit?: string
  score: number
  screenshots?: string[]  // Discord CDN URLs
  videos?: string[]
}
```

`catalog/guild/guild_items.json` — nested object:
```
{ [category]: { translations, [subCategory]: { translations, items: [ { name, translations, filename, tags, styles, hasVariants, category, subCategory } ] } } }
```
