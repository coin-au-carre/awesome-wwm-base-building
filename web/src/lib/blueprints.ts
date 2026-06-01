import { readFileSync } from "fs"
import { resolve } from "path"
import type { Blueprint, RankedBlueprint } from "@/types/blueprint"
import { slugify } from "@/lib/format"

function repoFile(relativePath: string) {
  return resolve(process.cwd(), "..", relativePath)
}

function loadJSON(): Blueprint[] {
  try {
    const raw = readFileSync(repoFile("data/blueprints.json"), "utf-8")
    return JSON.parse(raw)
  } catch {
    return []
  }
}

const PINNED_LAST = new Set(["Beautiful stick"])
const SCORE_OVERRIDES: Record<string, number> = { "Beautiful stick": 1 }

function sortByScore(items: Blueprint[]): RankedBlueprint[] {
  const main = [...items].filter((bp) => !PINNED_LAST.has(bp.name))
  const pinned = [...items].filter((bp) => PINNED_LAST.has(bp.name))
  const sorted = main.sort((a, b) => b.score - a.score)
  let rank = 1
  const ranked = sorted.map((bp, i) => {
    if (i > 0 && bp.score < sorted[i - 1].score) {
      rank = i + 1
    }
    return { ...bp, slug: slugify(bp.name), rank }
  })
  const lastRank = ranked.length + 1
  return [
    ...ranked,
    ...pinned.map((bp) => ({ ...bp, score: SCORE_OVERRIDES[bp.name] ?? bp.score, slug: slugify(bp.name), rank: lastRank })),
  ]
}

function collectTags(items: Blueprint[]): string[] {
  const tagSet = new Set<string>()
  items.forEach((bp) => bp.tags?.forEach((t) => tagSet.add(t)))
  return [...tagSet].sort()
}

const ALL_BLUEPRINTS: Blueprint[] = loadJSON()
const RANKED_BLUEPRINTS = sortByScore(ALL_BLUEPRINTS)
const BLUEPRINT_TAGS = collectTags(ALL_BLUEPRINTS)

export function getBlueprintsSortedByScore(): RankedBlueprint[] { return RANKED_BLUEPRINTS }
export function getBlueprintBySlug(slug: string): RankedBlueprint | undefined {
  return RANKED_BLUEPRINTS.find((bp) => bp.slug === slug)
}
export function getAllBlueprintTags(): string[] { return BLUEPRINT_TAGS }
export function hasBlueprints(): boolean { return ALL_BLUEPRINTS.length > 0 }

export function getLatestBlueprintsWithScreenshots(n: number): RankedBlueprint[] {
  return [...ALL_BLUEPRINTS]
    .reverse()
    .filter((bp) => bp.screenshots && bp.screenshots.length > 0)
    .slice(0, n)
    .map((bp) => RANKED_BLUEPRINTS.find((r) => r.discordThread === bp.discordThread))
    .filter((bp): bp is RankedBlueprint => bp !== undefined)
}

export function getRecentBlueprintsWithScreenshots(days: number, max: number): RankedBlueprint[] {
  const cutoff = Date.now() - days * 86400000
  return [...ALL_BLUEPRINTS]
    .reverse()
    .filter((bp) => {
      if (!bp.screenshots || !bp.screenshots.length) { return false }
      if (!bp.createdAt) { return false }
      const ms = new Date(bp.createdAt.replace(" at ", " ")).getTime()
      return !isNaN(ms) && ms >= cutoff
    })
    .slice(0, max)
    .map((bp) => RANKED_BLUEPRINTS.find((r) => r.discordThread === bp.discordThread))
    .filter((bp): bp is RankedBlueprint => bp !== undefined)
}
