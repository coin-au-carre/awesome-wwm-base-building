import { readFileSync, statSync } from "fs"
import type { Guild, RankedGuild } from "@/types/guild"
import { slugify } from "@/lib/slugify"

function loadJSON(filename: string): Guild[] {
  try {
    const raw = readFileSync(new URL(`../../../${filename}`, import.meta.url), "utf-8")
    return JSON.parse(raw)
  } catch {
    return []
  }
}

const ALL_GUILDS: Guild[] = loadJSON("guilds.json")
const ALL_SOLOS: Guild[] = loadJSON("solos.json")

export function getGuildsSortedByScore(): RankedGuild[] {
  const sorted = [...ALL_GUILDS].sort((a, b) => b.score - a.score)
  let rank = 1
  return sorted.map((g, i) => {
    if (i > 0 && g.score < sorted[i - 1].score) rank = i + 1
    return { ...g, slug: slugify(g.name), rank }
  })
}

export function getGuildBySlug(slug: string): RankedGuild | undefined {
  return getGuildsSortedByScore().find((g) => g.slug === slug)
}

export function getAllTags(): string[] {
  const tagSet = new Set<string>()
  ALL_GUILDS.forEach((g) => g.tags?.forEach((t) => tagSet.add(t)))
  return [...tagSet].sort()
}

export function getSolosSortedByScore(): RankedGuild[] {
  const sorted = [...ALL_SOLOS].sort((a, b) => b.score - a.score)
  let rank = 1
  return sorted.map((g, i) => {
    if (i > 0 && g.score < sorted[i - 1].score) rank = i + 1
    return { ...g, slug: slugify(g.name), rank }
  })
}

export function getSoloBySlug(slug: string): RankedGuild | undefined {
  return getSolosSortedByScore().find((g) => g.slug === slug)
}

export function getAllSoloTags(): string[] {
  const tagSet = new Set<string>()
  ALL_SOLOS.forEach((g) => g.tags?.forEach((t) => tagSet.add(t)))
  return [...tagSet].sort()
}

export function hasSolos(): boolean {
  return ALL_SOLOS.length > 0
}

export function getLastSyncDate(): string {
  try {
    const stat = statSync(new URL("../../../guilds.json", import.meta.url))
    return stat.mtime.toLocaleDateString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
    })
  } catch {
    return "Unknown"
  }
}
