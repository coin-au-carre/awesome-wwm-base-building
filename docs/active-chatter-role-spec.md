# Active Chatter Role — Spec

## Goal
A separate, manually-run command (`task active-roles`, not part of `task sync`)
that scans chat activity across selected channels and grants a Discord role to
users who have participated actively — without re-scanning messages or
re-hitting the Discord API for users who already have the role.

## Role name
**"Wanderer"**

## Tracked channels

| Channel ID | Name |
|---|---|
| 1483447711499030633 | 💬 chat-wwm |
| 1521760524235309191 | 🥑 homestead-system |
| 1514776235299967017 | 🏯 guild-building |
| 1514776286260756480 | 🏠 solo-building |
| 1483483683456286911 | 🔧 construction-help |
| 1522001435187744901 | 🧱 newbie-corner |
| 1483447711499030634 | 💡 tips-and-tricks |
| 1483451090048520252 | 🎲 whatever-showcase |

## Threshold
`>= 50` messages total across the tracked channels (combined, not per-channel).
_Tweak: raise/lower, or require a minimum number of distinct channels posted in._

## Activity window
All-time count (messages accumulate forever, no decay).
_Alternative: only count messages from the last N days — would need storing
timestamps instead of just a running count, more state to keep._

## State file: `data/chat_activity.json`
```jsonc
{
  "lastMessageID": {
    "1483447711499030633": "1509999999999999999",
    // ... one per tracked channel, so reruns only fetch new messages
  },
  "counts": {
    "207164708199989248": 42
    // userID -> lifetime message count across tracked channels
  }
}
```

## Role assignment
Reuses the existing `RoleCache` (`internal/discord/roles.go`,
`data/role_assignments.json`) so a user already holding the role is never
re-checked against Discord's API on subsequent runs — same pattern as
`cmd/assign-roles`.

## Command
New `cmd/assign-active-roles`, following the shape of `cmd/assign-roles`:
1. Load `data/chat_activity.json` (or start empty).
2. For each tracked channel, page through `ChannelMessages(..., after: lastMessageID)`,
   bump `counts[authorID]`, skip bots, update `lastMessageID`.
3. Save `data/chat_activity.json`.
4. For every user with `counts[user] >= threshold`, assign the role via
   `RoleCache`-backed helper (skips already-assigned users, no API call).

## Role ID
`1525608229613338634` ("Wanderer") — hardcoded as a const in
`cmd/assign-active-roles/main.go`. Role IDs don't change, so no env var either
way, even if this later becomes a scheduled GitHub Action.

`BASE_BUILDER_ROLE_ID` and `SOLO_BUILDER_ROLE_ID` (used for the "builders who
aren't active chatters" report) already exist in `.env` — read from there as
usual via `cmdutil.RequireEnv`, not hardcoded.

## Run cadence
Manual for now: `task active-roles`, run whenever you feel like it. May become
a scheduled GitHub Action later (weekly), similar to `sync-homestead.yml`.

## Bots
Excluded — `discordgo.Message.Author.Bot` is checked and skipped, so Ruby and
any other bots never accumulate counts.

## Builders who aren't active chatters
The command also prints (stdout, not a Discord role) the set of users who
hold the `BASE_BUILDER_ROLE_ID` or `SOLO_BUILDER_ROLE_ID` role (via
`data/role_assignments.json`, already keyed by role ID) but whose
`counts[userID] < threshold` — i.e. builders who show up in `guilds.json` /
`solos.json` but rarely post in the tracked chat channels. Free to compute
once `counts` exists, no extra Discord calls needed.

## Performance: first run scans full channel history
Yes, the first run is the slow one — every run after it is cheap because of
`lastMessageID`.

- Discord returns 100 messages per `ChannelMessages` call, so a channel with
  20k messages is ~200 requests. Across 8 channels that's low thousands of
  requests total.
- Rate limit is generous for this endpoint (roughly 50 req/s per route), so
  the bottleneck is really just network round-trips, not throttling.
- Realistic estimate: a few minutes for the very first backfill across all 8
  channels, then each later run only fetches messages posted since the last
  run (seconds).
- If that first run is a concern, it can run once manually/off-hours; no need
  to make it fast since it only happens once.

## Open questions for you to tweak
- Threshold number (50, tweak once counts are real).
