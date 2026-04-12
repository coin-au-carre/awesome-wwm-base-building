import { readFileSync } from "fs"
import type { Guild, RankedGuild } from "@/types/guild"
import { slugify, formatBuilderName } from "@/lib/slugify"

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
    return { ...g, slug: slugify(g.name), rank }
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
    .filter((g) => g.screenshots && g.screenshots.length > 0)
    .slice(0, n)
    .map((g) => ranked.find((r) => r.name === g.name))
    .filter((g): g is RankedGuild => g !== undefined)
  return withShots
}

export function getLastSyncDate(): string {
  try {
    const raw = readFileSync(new URL("../../../data/last_sync.json", import.meta.url), "utf-8")
    const { syncedAt } = JSON.parse(raw)
    return new Date(syncedAt).toLocaleString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      timeZone: "UTC",
      timeZoneName: "short",
    })
  } catch {
    return "Unknown"
  }
}
