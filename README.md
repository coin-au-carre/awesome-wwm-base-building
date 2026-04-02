# awesome-wwm-base-building

Community showcase of guild bases and solo builds for [Where Winds Meet](https://wherethewwindsmeet.com/).

**Live site:** https://coin-au-carre.github.io/awesome-wwm-base-building

---

## Stack

- **Go 1.26** — Discord sync bot (`cmd/`, `internal/`)
- **Astro 5 + shadcn/ui + Tailwind 4** — static website (`web/`)
- **GitHub Actions** — `sync.yml` (data sync, several times/day) + `deploy.yml` (site deploy on push)

## Quick start

```sh
cp .env.example .env   # fill in Discord tokens
task sync              # crawl Discord → update guilds.json / solos.json
task web               # start Astro dev server at http://localhost:4321
```

See `task help` for all available commands, and `AGENTS.md` for full architecture docs.

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `RUBY_BOT_TOKEN` | ✅ | Discord bot token |
| `GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID` | ✅ | Guild forum channel ID |
| `SOLO_BUILD_SHOWCASE_CHANNEL_FORUM_ID` | — | Solo builds forum channel ID |
| `BOT_CHANNEL_ID` | — | Channel for bot notifications |
| `BASE_BUILDER_ROLE_ID` | — | Discord role for builders |
| `BASE_CRITIC_ROLE_ID` | — | Discord role for voters |
| `ANTHROPIC_API_KEY` | — | Claude AI (bot feature) |
