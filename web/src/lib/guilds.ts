import { readFileSync } from "fs"
import { resolve } from "path"
import type { Guild, RankedGuild } from "@/types/guild"
import { slugify, formatBuilderName } from "@/lib/format"
import { isBuilderSubmission } from "@/lib/config"

export interface GuildRedirect {
  fromSlug: string
  toSlug: string
}

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
  type?: string
  scheduledStart: string
  scheduledEnd?: string
  location?: string
  channelType?: "voice" | "stage"
  channelName?: string
  status: string
  subscriberCount?: number
  discordUrl: string
  image?: string
}

export { formatBuilderName }

// process.cwd() = web/ at build time; data files live one level up at repo root
function repoFile(relativePath: string) {
  return resolve(process.cwd(), "..", relativePath)
}

function loadJSON(filename: string): Guild[] {
  try {
    const raw = readFileSync(repoFile(filename), "utf-8")
    return JSON.parse(raw)
  } catch {
    return []
  }
}

function guildSlug(g: Guild): string {
  const base = slugify(g.guildName ?? g.name)
  return g.buildTitle ? `${base}-${slugify(g.buildTitle)}` : base
}

function sortByScore(items: Guild[]): RankedGuild[] {
  const sorted = [...items].sort((a, b) => b.score - a.score)
  let rank = 1
  const seen = new Map<string, number>()
  return sorted.map((g, i) => {
    if (i > 0 && g.score < sorted[i - 1].score) {
      rank = i + 1
    }
    const base = guildSlug(g)
    const n = seen.get(base) ?? 0
    seen.set(base, n + 1)
    const slug = n === 0 ? base : `${base}-${n + 1}`
    return { ...g, slug, rank }
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
export function getGuildBuilds(name: string): RankedGuild[] {
  return RANKED_GUILDS.filter((g) => g.name === name)
}

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
    .filter((g) => g.screenshots && g.screenshots.length > 0 && isBuilderSubmission(g))
    .slice(0, n)
    .map((g) => ranked.find((r) => r.discordThread === g.discordThread))
    .filter((g): g is RankedGuild => g !== undefined)
  return withShots
}

export function getRecentGuildsWithScreenshots(days: number): RankedGuild[] {
  const cutoff = Date.now() - days * 86400000
  return [...ALL_GUILDS]
    .reverse()
    .filter((g) => {
      if (!g.screenshots || !g.screenshots.length || !isBuilderSubmission(g)) { return false }
      if (!g.createdAt) { return false }
      const ms = new Date(g.createdAt.replace(" at ", " ")).getTime()
      return !isNaN(ms) && ms >= cutoff
    })
    .map((g) => RANKED_GUILDS.find((r) => r.discordThread === g.discordThread))
    .filter((g): g is RankedGuild => g !== undefined)
}

export function getLatestSolosWithScreenshots(n: number): RankedGuild[] {
  const ranked = RANKED_SOLOS
  const withShots = [...ALL_SOLOS]
    .reverse()
    .filter((g) => g.screenshots && g.screenshots.length > 0)
    .slice(0, n)
    .map((g) => ranked.find((r) => r.discordThread === g.discordThread))
    .filter((g): g is RankedGuild => g !== undefined)
  return withShots
}

export function getHiddenGems(): RankedGuild[] {
  try {
    const raw = readFileSync(repoFile("data/hidden_gems.json"), "utf-8")
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

/** Returns a direct link to the builder's guild or solo page, or null if not found. Guilds takes priority over solos. */
export function getBuilderSearchPath(name: string): string | null {
  const norm = (s: string) => s.normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLowerCase()
  const n = norm(name)
  const guild = RANKED_GUILDS.find((g) => (g.builders ?? []).some((b) => norm(formatBuilderName(b)).includes(n)))
  if (guild) { return `/guilds/${guild.slug}` }
  const solo = RANKED_SOLOS.find((g) => (g.builders ?? []).some((b) => norm(formatBuilderName(b)).includes(n)))
  if (solo) { return `/solos/${solo.slug}` }
  return null
}

export const UPCOMING_EVENTS_WINDOW_MS = 48 * 60 * 60 * 1000
export const FEATURED_EVENTS_WINDOW_MS = 24 * 60 * 60 * 1000

export function getUpcomingEvents(): GameEvent[] {
  try {
    const raw = readFileSync(repoFile("data/events.json"), "utf-8")
    const events: GameEvent[] = JSON.parse(raw)
    const now = Date.now()
    const cutoff = now + UPCOMING_EVENTS_WINDOW_MS
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
    const raw = readFileSync(repoFile("data/reactions.json"), "utf-8")
    return JSON.parse(raw)
  } catch {
    return {}
  }
}

export function getUsers(): UserMap {
  try {
    const base: UserMap = JSON.parse(readFileSync(repoFile("data/users.json"), "utf-8"))
    try {
      const extra: UserMap = JSON.parse(readFileSync(repoFile("data/additionnal_users.json"), "utf-8"))
      return { ...base, ...extra }
    } catch {
      return base
    }
  } catch {
    return {}
  }
}

export function getVoterBlacklist(): string[] {
  try {
    const raw = readFileSync(repoFile("data/voter_blacklist.json"), "utf-8")
    return JSON.parse(raw)
  } catch {
    return []
  }
}

function computeRedirects(ranked: RankedGuild[]): GuildRedirect[] {
  const redirects: GuildRedirect[] = []
  const allSlugs = new Set(ranked.map((g) => g.slug))

  const groups = new Map<string, RankedGuild[]>()
  for (const g of ranked) {
    const list = groups.get(g.name) ?? []
    list.push(g)
    groups.set(g.name, list)
  }

  for (const builds of groups.values()) {
    const current = builds.find((b) => b.isCurrent) ?? (builds.length === 1 ? builds[0] : undefined)
    if (!current || !current.buildTitle) { continue }

    // guildName slug (without buildTitle) → current build slug
    const baseSlug = slugify(current.guildName ?? current.name)
    if (baseSlug !== current.slug && !allSlugs.has(baseSlug)) {
      redirects.push({ fromSlug: baseSlug, toSlug: current.slug })
    }
  }

  return redirects
}

export function getGuildRedirects(): GuildRedirect[] { return computeRedirects(RANKED_GUILDS) }
export function getSoloRedirects(): GuildRedirect[] { return computeRedirects(RANKED_SOLOS) }

export function getLastSyncDate(): string {
  try {
    const raw = readFileSync(repoFile("data/last_sync.json"), "utf-8")
    const { syncedAt } = JSON.parse(raw)
    return new Date(syncedAt).toISOString()
  } catch {
    return ""
  }
}
