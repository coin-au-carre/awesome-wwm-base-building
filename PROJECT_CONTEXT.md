# Project Context: Where Builders Meet

A community showcase for guild bases and solo builds in **Where Winds Meet**, an open-world RPG set in ancient China (Jiang Hu). The game has no in-game gallery, and great screenshots tend to disappear in Discord threads. This project solves that.

## What it is

A public website where players can browse, vote on, and get inspired by guild bases and solo builds submitted by the community. It is community-run, not affiliated with the game developers.

Website: deployed on GitHub Pages under `/awesome-wwm-base-building`. Real website is https://www.wherebuildersmeet.com.

## How submissions work

Players submit their guild base or solo build by posting in dedicated forum channels on the community Discord server (`discord.gg/Qygt9u26Bn`):

- `#guild-base-showcase` for guild bases (multi-player guild headquarters)
- `#solo-build-showcase` for individual solo builds

A submission thread should include screenshots (Discord CDN links), an optional lore writeup, and an optional "what to visit" guide describing highlights of the base.

## How scoring works

Community members vote by reacting to the first post of a submission thread:

| Reaction | Points | Meaning |
|----------|--------|---------|
| ⭐ | +2 | Best overall |
| 👍 / 👍🏻–👍🏿 | +1 | Good base |
| 🔥 | +1 | Amazing creativity |
| ❤️ | +1 | Love it |

Bonus points:
- +1 if the builder wrote a **lore** section
- +1 if the builder wrote a **what to visit** section

### Voter weights

To keep rankings fair and prevent coordinated boosting, votes are weighted by how broadly a voter engages with the community:

| Distinct threads reacted to | Weight multiplier |
|-----------------------------|-------------------|
| 4+ | ×1 (baseline) |
| 8+ | ×2 |
| 12+ | ×3 (Critic) |

A voter who reacts to fewer than 4 threads has weight ×0 — their votes don't count until they engage broadly. Weights are computed **per channel**: guild reactions only determine guild voter weights, solo reactions only determine solo voter weights.

A voter who reacts to many different submissions counts more than someone who voted on only one or two. This discourages guild leaders from asking members to vote only for their own base.

### Blacklist

Voters identified as abusing the system (e.g. voting strategically only to inflate one guild's score) can be added to a blacklist (`data/voter_blacklist.json`). Blacklisted voters are excluded from scoring entirely. Their reactions are still stored in `reactions.json` so the effect of the blacklist can be toggled on the admin page.

## Discord roles

Two roles are automatically assigned by the bot:

- **Builder** (🏗️): awarded to users who have posted a guild base in `#guild-base-showcase`
- **Critic** (🗳️): awarded to active community voters who engage broadly across many guilds

Two roles are manually assigned by moderators:

- **Trusted Eye**: granted to community members recognized for their expertise and quality feedback on guild bases
- **Guild Cartographer**: granted to members who contribute significantly to mapping and documenting guild bases

## How the site stays updated

A Go bot crawls the Discord forum channels several times a day. It reads all thread metadata and reactions, computes scores, and writes the results to `guilds.json` and `solos.json`. A GitHub Actions workflow then rebuilds and redeploys the static site automatically. No manual curation is needed.

## What the site shows

- **Homepage**: top-rated guild bases with a photo showcase (one random screenshot from each of the top 9 ranked guilds), plus a full ranked table of all submitted bases with tag filtering and search.
- **Guild detail page**: screenshots gallery, lore, what-to-visit, builder name(s), score breakdown.
- **Solo builds**: same structure as guilds, for individual player builds.
- **Item catalog**: a browsable catalog of in-game building items (floors, walls, roofs, furniture, etc.) with variants, tags, styles, components, and buy locations. Fully launched.
- **Tutorials**: written guides and video tutorials (YouTube and TikTok) to help players build better. Items are tagged by difficulty (Beginner / Advanced) and grouped into columns. Guides are Markdown content collection entries; videos are listed in a `lib/videos.ts` file.
- **Events** (alpha): upcoming in-game player events organised from guild bases. Synced from Discord Scheduled Events. Only Builder and Critic role holders can create events. Displays type, guild, date/time, location, attendee count, and a link to RSVP on Discord.
- **How it works**: explanation of the scoring and voting system (what you are reading is derived from that page).
- **Contribute**: instructions for builders (how to submit), voters (how to vote), and developers (how to contribute to the project).

## Community tone and purpose

This is a fan-made, volunteer-run project. The goal is to celebrate creativity in Where Winds Meet, make great builds discoverable, and give builders recognition. The community Discord is welcoming to new players and veteran builders alike.
