import { readFileSync } from "fs"
import type { Guild, RankedGuild } from "@/types/guild"
import { slugify, formatBuilderName } from "@/lib/format"
import { MOD_IDS } from "@/lib/config"

export interface UserInfo {
  username: string
  globalName?: string
  nickname?: string
}
export type ReactionMap = Record<string, Record<string, string[]>> // threadID → emoji → userID[]
export type UserMap = Record<string, UserInfo>

export interface GameEvent {
  id: string
  name: string
  description?: string
  guildName?: string
  guildId?: string
  scheduledStart: string
  scheduledEnd?: string
  location?: string
  status: string
  subscriberCount?: number
  discordUrl: string
  image?: string
}

export { formatBuilderName }

function loadJSON(filename: string): Guild[] {
  try {
    const raw = readFileSync(new URL(`../../../${filename}`, import.meta.url), "utf-8")
    return JSON.parse(raw)
  } catch {
    return []
  }
}

function sortByScore(items: Guild[]): RankedGuild[] {
  const sorted = [...items].sort((a, b) => b.score - a.score)
  let rank = 1
  return sorted.map((g, i) => {
    if (i > 0 && g.score < sorted[i - 1].score) {
      rank = i + 1
    }
    return { ...g, slug: slugify(g.guildName ?? g.name), rank }
  })
}

function collectTags(items: Guild[]): string[] {
  const tagSet = new Set<string>()
  items.forEach((g) => g.tags?.forEach((t) => tagSet.add(t)))
  return [...tagSet].sort()
}

const ALL_GUILDS: Guild[] = loadJSON("data/guilds.json")
const ALL_SOLOS: Guild[] = loadJSON("data/solos.json")

const RANKED_GUILDS = sortByScore(ALL_GUILDS)
const RANKED_SOLOS = sortByScore(ALL_SOLOS)
const GUILD_TAGS = collectTags(ALL_GUILDS)
const SOLO_TAGS = collectTags(ALL_SOLOS)

export function getGuildsSortedByScore(): RankedGuild[] { return RANKED_GUILDS }
export function getGuildBySlug(slug: string): RankedGuild | undefined {
  return RANKED_GUILDS.find((g) => g.slug === slug)
}
export function getAllTags(): string[] { return GUILD_TAGS }

export function getSolosSortedByScore(): RankedGuild[] { return RANKED_SOLOS }
export function getSoloBySlug(slug: string): RankedGuild | undefined {
  return RANKED_SOLOS.find((g) => g.slug === slug)
}
export function getAllSoloTags(): string[] { return SOLO_TAGS }

export function hasSolos(): boolean { return ALL_SOLOS.length > 0 }

export function getLatestGuildsWithScreenshots(n: number): RankedGuild[] {
  const ranked = RANKED_GUILDS
  const withShots = [...ALL_GUILDS]
    .reverse()
    .filter((g) => g.screenshots && g.screenshots.length > 0 && (g.postedOnBehalfOf || !MOD_IDS.has(g.posterDiscordId ?? "")))
    .slice(0, n)
    .map((g) => ranked.find((r) => r.name === g.name))
    .filter((g): g is RankedGuild => g !== undefined)
  return withShots
}

export function getLatestSolosWithScreenshots(n: number): RankedGuild[] {
  const ranked = RANKED_SOLOS
  const withShots = [...ALL_SOLOS]
    .reverse()
    .filter((g) => g.screenshots && g.screenshots.length > 0)
    .slice(0, n)
    .map((g) => ranked.find((r) => r.name === g.name))
    .filter((g): g is RankedGuild => g !== undefined)
  return withShots
}

export function getHiddenGems(): RankedGuild[] {
  try {
    const raw = readFileSync(new URL("../../../data/hidden_gems.json", import.meta.url), "utf-8")
    const entries: string[] = JSON.parse(raw)
    return entries
      .map((entry) =>
        RANKED_GUILDS.find(
          (g) => g.slug === entry || g.name === entry || g.guildName === entry
        )
      )
      .filter((g): g is RankedGuild => g !== undefined)
  } catch {
    return []
  }
}

/** Returns the search URL for a builder name, or null if not found in either dataset. Guilds takes priority over solos. */
export function getBuilderSearchPath(name: string): string | null {
  const norm = (s: string) => s.normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLowerCase()
  const n = norm(name)
  const inGuilds = ALL_GUILDS.some((g) => (g.builders ?? []).some((b) => norm(formatBuilderName(b)).includes(n)))
  if (inGuilds) { return `/?q=${encodeURIComponent(name)}` }
  const inSolos = ALL_SOLOS.some((g) => (g.builders ?? []).some((b) => norm(formatBuilderName(b)).includes(n)))
  if (inSolos) { return `/solo?q=${encodeURIComponent(name)}` }
  return null
}

export function getUpcomingEvents(): GameEvent[] {
  try {
    const raw = readFileSync(new URL("../../../data/events.json", import.meta.url), "utf-8")
    const events: GameEvent[] = JSON.parse(raw)
    const now = Date.now()
    const cutoff = now + 24 * 60 * 60 * 1000
    return events
      .filter((e) => {
        const start = new Date(e.scheduledStart).getTime()
        const end = e.scheduledEnd ? new Date(e.scheduledEnd).getTime() : start
        const isLive = start <= now && end >= now
        const isUpcoming = start > now && start <= cutoff
        return e.status === "active" || isLive || isUpcoming
      })
      .sort((a, b) => new Date(a.scheduledStart).getTime() - new Date(b.scheduledStart).getTime())
  } catch {
    return []
  }
}

export function getReactions(): ReactionMap {
  try {
    const raw = readFileSync(new URL("../../../data/reactions.json", import.meta.url), "utf-8")
    return JSON.parse(raw)
  } catch {
    return {}
  }
}

export function getUsers(): UserMap {
  try {
    const raw = readFileSync(new URL("../../../data/users.json", import.meta.url), "utf-8")
    return JSON.parse(raw)
  } catch {
    return {}
  }
}

export function getVoterBlacklist(): string[] {
  try {
    const raw = readFileSync(new URL("../../../data/voter_blacklist.json", import.meta.url), "utf-8")
    return JSON.parse(raw)
  } catch {
    return []
  }
}

export function getLastSyncDate(): string {
  try {
    const raw = readFileSync(new URL("../../../data/last_sync.json", import.meta.url), "utf-8")
    const { syncedAt } = JSON.parse(raw)
    return new Date(syncedAt).toISOString()
  } catch {
    return ""
  }
}
