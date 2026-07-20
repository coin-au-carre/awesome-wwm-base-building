# Builder identity system

> Design doc, not yet implemented. Written to prepare the work, not to
> describe what exists today.

## The problem

WBM currently has **two disconnected notions of "who a builder is"**, and
neither knows anything about a builder's actual NetEase account:

- **Web** (`web/src/lib/builders.ts` + `web/src/lib/builder-aliases.ts`): a
  builder is a **slug parsed from free-text display names**
  (`builderSlug(formatBuilderName(rawName))`), with a hand-maintained
  `BUILDER_ALIASES` map folding known misspelled variants onto one canonical
  slug. `getBuilderProfile(slug)` aggregates guild bases, solo bases,
  blueprints, and homestead credits whose contributor name resolves to that
  slug. No Discord ID, no NetEase ID anywhere in this model.
- **Discord bot** (`internal/discord/builder.go`'s `/builder` command):
  keyed by the **real Discord user ID**, matched against `PosterDiscordID`
  on guild/solo entries. No name/alias resolution at all.

These diverge structurally. A builder unified across spelling variants on
the web might still be several different "people" bot-side if their
Discord ID isn't attached to every submission, and vice versa.

Separately, we already have real NetEase `number_id`s sitting
**unstructured inside free text** in `data/guilds.json`/`data/solos.json` —
embedded in `builders[]` display strings (`"Ðìana (uid 2039668966)"`,
`"Hantiya [2039322781]"`, `"Airisan(UID:0023243021)"`, etc.) and
occasionally in `lore` (`"...kindly pm me UID 4042165548..."`). Builders are
already doing this themselves, unprompted — the demand for "let people
attach their in-game account" clearly already exists, just with no
structured place to put it.

## Proposed data model

One canonical record per real person, in a new `data/builder_identities.json`:

```ts
{
  discordId: string            // stable anchor — every guild/solo submission already carries posterDiscordId
  canonicalSlug: string        // today's builderSlug, kept as the display/URL identity — user-editable via /wwm-uid, must be unique across all records
  aliasSlugs: string[]         // absorbs today's BUILDER_ALIASES reverse-lookup
  ingameNickname?: string      // current NetEase display name — derived only, never typed by a user; overwrite on drift, no history kept
  neteaseNumberId?: string     // public-facing NetEase account number (already shown to every player in-game) — the only NetEase-side field a user actually types in
  neteasePid?: string          // NetEase's internal player id, resolved once via find_people/by_number_id
  neteaseHostnum?: number      // server-shard int paired with pid — required alongside it for designer/plan-batch calls
}
```

**Includes `pid`/`hostnum` directly, on purpose.** `wbm-relay`'s existing
convention (`pkg/relay/designer.go`) never exposes them to the frontend,
but that's an API-cleanliness choice, not a security boundary — `pid`/
`hostnum` aren't secret, since anyone can derive them from a plain
`number_id` via one unauthenticated `find_people/by_number_id` call.
Storing them here once, at registration time, means every later lookup —
by the web, by `wbm-relay`, by the future task command in Piece 3 — reads
straight from this file instead of re-resolving live. One file, one source
of truth, no second private cache to keep in sync.

## Piece 1 — `builder-aliases.ts` → JSON refactor

Move `BUILDER_ALIASES`'s data into `data/builder_identities.json` (the
`aliasSlugs` field above). Keep `resolveCanonical(slug)` and
`getAllSlugsForCanonical(slug)`'s existing signatures **unchanged** in
`builder-aliases.ts` — only their implementation changes, to read from the
new JSON instead of an inline object literal.

This keeps the refactor low-risk: every current importer keeps working
unmodified since the function contracts don't change. Confirmed importers
today: `web/src/lib/builders.ts`, `web/src/lib/mentions.ts`,
`web/src/components/LeaderboardTable.tsx`,
`web/src/components/BlueprintGrid.tsx`,
`web/src/components/TutorialsFilter.tsx`,
`web/src/components/BuilderLinks.astro`, and the `[slug].astro` pages for
builders, blueprints, solos, guilds, tutorials, plus `homestead.astro` and
`credits.astro`.

## Piece 2 — Discord self-service command: `/wwm-uid`

Push-based. There's no existing precedent for this in the bot today —
`homestead-sync` is pull-based (it infers membership from Discord role
assignments, not from a user submitting data about themself). No command
option, though — instead:

1. Running `/wwm-uid` opens a **modal** (same pattern already used by
   `scout-guild`/`submit-guild` in `submit.go`: `discordgo.InteractionResponseModal` +
   `TextInput`, routed back on `InteractionModalSubmit` by `CustomID`) with
   **two fields**, each pre-filled from the caller's existing record if one
   exists, empty otherwise:
   - **"Builder Name"** — the `canonicalSlug`. Editable because it's the
     public URL identity (`/builders/<slug>`) and people do want to fix a
     bad auto-generated slug; **must be unique**, checked against every
     other record's `canonicalSlug` on submit (excluding the caller's own
     existing entry) — reject with a clear error and let them retry if
     it's already taken by someone else.
   - **"Your In-Game UID"** — the `neteaseNumberId`, **not required** (an
     empty submission clears it, same as before).
   `ingameNickname` is **not a modal field at all** — it's never typed,
   only ever derived server-side from resolving the UID, so there's
   nothing for the user to edit or get wrong there.
2. **Validate before saving.**
   - **Slug taken by someone else**: reject immediately, before touching
     the UID at all — ask the caller to resubmit with a different one.
   - **UID field left empty**: remove `neteaseNumberId`/`neteasePid`/
     `neteaseHostnum`/`ingameNickname` from the record entirely (absent,
     not blank — see Piece 3/4's reliance on that).
   - **UID unchanged** (resubmitted the same pre-filled value): skip
     straight to the confirmation step below, no need to re-hit the API.
   - **UID set/changed**: call `find_people/by_number_id` directly
     (confirmed to need no auth/secret) to resolve `pid`/`hostnum`/
     nickname.
3. **Confirm before saving, don't just display-and-save.** Since the
   nickname can only ever be *guessed* from the UID (never typed by the
   user), a typo'd UID could resolve to a real but wrong account. Reply
   with the resolved nickname plus **Confirm/"Not me" buttons**
   (`discordgo.MessageComponent`s, not just a text confirmation): Confirm
   writes the record; "Not me" drops the attempt and tells the caller to
   run `/wwm-uid` again rather than silently saving a wrong match.
4. **Write + commit directly from the live bot process.** This has a
   direct precedent already in this codebase: `data/streaming.json` is
   likewise owned and committed by the live, lock-holding bot process
   itself, not by a separate `cmd/*-sync` + GitHub Actions job. The same
   shape works here — no new persistence pattern needs inventing.

Left open for whoever implements this: exact command wording (beyond the
name itself), and whether *changing* an already-registered UID should stay
fully self-service or need a mod's approval, given it becomes a
trust-bearing field feeding public attribution.

## Piece 3 — Backfill task command for `number_id → pid/hostnum`

Piece 2 resolves and stores `pid`/`hostnum` at registration time for anyone
who goes through `/wwm-uid` going forward. But we already have
`number_id`s sitting **unstructured in free text** today (the
`builders[]`/`lore` examples in "The problem" above) — those need a
one-time backfill, not a live per-request resolve.

Proposed: a new `cmd/resolve-builder-ids/main.go` in **this** repo (not
`wbm-relay` — now that `pid`/`hostnum` live in the public data file, there's
no private-side piece left to build), mirroring `cmd/homestead-sync`'s
shape: load `data/builder_identities.json`, call `find_people/by_number_id`
only for entries still missing `neteasePid`/`neteaseHostnum`, write the
merged result back. Run manually at first; wire up a GitHub Actions
schedule later only if entries start going stale often enough to justify
it — most of them won't, since `pid` effectively never changes once
resolved.

`wbm-relay/pkg/relay/authorcache.go`'s existing in-memory,
24-hour-TTL cache (`cache.NewTTL[playerRef]`) still has its own job here:
it's `wbm-relay`'s live-request cache for numbers *not yet* in
`builder_identities.json` (e.g. designers who've never registered but show
up browsing the gallery) — this backfill command doesn't replace it, it
just means anyone who *has* registered skips the live resolve entirely,
both in `wbm-relay` and on the web.

**Already validated live** (this session, no auth needed for any of these
calls): resolving `2039322781`, `4035912421`, `2039668966`, `1022322056`,
and `2034205345` via `find_people/by_number_id` each took one call, and a
single `get_face_designer_brief_info_batch` call with all five resulting
`pid`s in one `pid_dict` returned every designer's plan-list stats
together — confirming the batch shape this backfill (and the eventual
Discord-ID-first `getBuilderProfile` merge in Piece 4) can build on.

## Piece 4 — Future: merging the two builder-profile systems

Not this pass. The natural merge once `data/builder_identities.json`
exists: `getBuilderProfile` becomes **Discord-ID-first** — guild/solo
entries already carry `posterDiscordId`, so match on that directly against
the identity file, and fall back to today's name-slug+alias resolution
only for entries with no attached Discord poster (scouted/legacy
submissions, or blueprint/homestead credit lines that are free text only,
with no Discord account attached at all). The alias system isn't discarded
by this — it's demoted to the fallback layer for exactly the content that
has no Discord ID to anchor on.

## Open questions

- Exact `/wwm-uid` command name/wording.
- Self-service vs. mod-gated re-registration of an existing mapping.
- Whether `ingameNickname` needs a refresh policy — it can drift
  even though `pid`/`hostnum` effectively don't, so a registered entry's
  name could go stale while its identity stays perfectly valid.
