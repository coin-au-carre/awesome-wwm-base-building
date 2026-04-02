import { readFileSync, statSync } from "fs"
import type { Guild, RankedGuild } from "@/types/guild"
import { slugify } from "@/lib/slugify"

// Resolved at build time: web/src/lib/ → web/ → repo root → guilds.json
const raw = readFileSync(
  new URL("../../../guilds.json", import.meta.url),
  "utf-8"
)
const ALL_GUILDS: Guild[] = JSON.parse(raw)

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
