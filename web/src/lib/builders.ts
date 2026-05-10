import type { RankedGuild } from "@/types/guild"
import type { RankedBlueprint } from "@/types/blueprint"
import { getGuildsSortedByScore, getSolosSortedByScore } from "@/lib/guilds"
import { getBlueprintsSortedByScore } from "@/lib/blueprints"
import { formatBuilderName, builderSlug } from "@/lib/format"

export interface BuilderProfile {
  name: string
  slug: string
  guilds: RankedGuild[]
  solos: RankedGuild[]
  blueprints: RankedBlueprint[]
}

function matchesSlug(rawName: string, targetSlug: string): boolean {
  const cleaned = formatBuilderName(rawName)
  return !!cleaned && builderSlug(cleaned) === targetSlug
}

export function getAllBuilderSlugs(): { name: string; slug: string }[] {
  const slugMap = new Map<string, string>() // slug → canonical name

  function addName(name: string) {
    const s = builderSlug(name)
    if (s && !slugMap.has(s)) slugMap.set(s, name)
  }

  for (const g of getGuildsSortedByScore()) {
    for (const b of g.builders ?? []) {
      const name = formatBuilderName(b)
      if (name) addName(name)
    }
  }
  for (const g of getSolosSortedByScore()) {
    for (const b of g.builders ?? []) {
      const name = formatBuilderName(b)
      if (name) addName(name)
    }
  }
  for (const bp of getBlueprintsSortedByScore()) {
    if (bp.builderName) addName(bp.builderName)
  }

  return [...slugMap.entries()].map(([slug, name]) => ({ name, slug }))
}

export interface NotableBuilder {
  name: string
  slug: string
  coverImage?: string
  guildCount: number
  soloCount: number
  blueprintCount: number
  tutorialCount: number
  typeCount: number
}

export function getNotableBuilders(
  tutorialCountBySlug?: Map<string, number>,
  limit?: number,
): NotableBuilder[] {
  const bySlug = new Map<string, NotableBuilder>()

  function get(slug: string, name: string): NotableBuilder {
    if (!bySlug.has(slug)) bySlug.set(slug, { name, slug, guildCount: 0, soloCount: 0, blueprintCount: 0, tutorialCount: 0, typeCount: 0 })
    return bySlug.get(slug)!
  }

  for (const g of getGuildsSortedByScore()) {
    for (const b of g.builders ?? []) {
      const name = formatBuilderName(b)
      if (!name) continue
      const s = builderSlug(name)
      if (!s) continue
      const entry = get(s, name)
      if (!entry.coverImage) entry.coverImage = g.coverImage ?? g.screenshots?.[0]
      entry.guildCount++
    }
  }

  for (const g of getSolosSortedByScore()) {
    for (const b of g.builders ?? []) {
      const name = formatBuilderName(b)
      if (!name) continue
      const s = builderSlug(name)
      if (!s) continue
      const entry = get(s, name)
      if (!entry.coverImage) entry.coverImage = g.coverImage ?? g.screenshots?.[0]
      entry.soloCount++
    }
  }

  for (const bp of getBlueprintsSortedByScore()) {
    if (!bp.builderName) continue
    const s = builderSlug(bp.builderName)
    if (!s) continue
    const entry = get(s, bp.builderName)
    if (!entry.coverImage) entry.coverImage = bp.coverImage ?? bp.screenshots?.[0]
    entry.blueprintCount++
  }

  if (tutorialCountBySlug) {
    for (const [s, count] of tutorialCountBySlug) {
      if (bySlug.has(s)) bySlug.get(s)!.tutorialCount = count
    }
  }

  for (const entry of bySlug.values()) {
    entry.typeCount = [entry.guildCount, entry.soloCount, entry.blueprintCount, entry.tutorialCount].filter(Boolean).length
  }

  return [...bySlug.values()]
    .filter((e) => e.typeCount >= 2)
    .sort((a, b) => {
      // 1. most diverse contributors first
      if (b.typeCount !== a.typeCount) return b.typeCount - a.typeCount
      // 2. tutorial authors surface before pure builders
      if (b.tutorialCount !== a.tutorialCount) return b.tutorialCount - a.tutorialCount
      // 3. total contributions as tiebreaker
      return (b.guildCount + b.soloCount + b.blueprintCount) - (a.guildCount + a.soloCount + a.blueprintCount)
    })
    .slice(0, limit)
}

export function getActiveBuilderSlugs(tutorialAuthorSlugs?: Set<string>): Set<string> {
  const typeCount = new Map<string, number>()

  function add(slug: string) {
    typeCount.set(slug, (typeCount.get(slug) ?? 0) + 1)
  }

  const seenGuild = new Set<string>()
  for (const g of getGuildsSortedByScore()) {
    for (const b of g.builders ?? []) {
      const s = builderSlug(formatBuilderName(b))
      if (s && !seenGuild.has(s)) { seenGuild.add(s); add(s) }
    }
  }

  const seenSolo = new Set<string>()
  for (const g of getSolosSortedByScore()) {
    for (const b of g.builders ?? []) {
      const s = builderSlug(formatBuilderName(b))
      if (s && !seenSolo.has(s)) { seenSolo.add(s); add(s) }
    }
  }

  const seenBlueprint = new Set<string>()
  for (const bp of getBlueprintsSortedByScore()) {
    if (!bp.builderName) continue
    const s = builderSlug(bp.builderName)
    if (s && !seenBlueprint.has(s)) { seenBlueprint.add(s); add(s) }
  }

  if (tutorialAuthorSlugs) {
    for (const s of tutorialAuthorSlugs) {
      if (s) add(s)
    }
  }

  const active = new Set<string>()
  for (const [s, count] of typeCount) {
    if (count >= 2) active.add(s)
  }
  return active
}

export function getBuilderProfile(slug: string): BuilderProfile | null {
  const guilds = getGuildsSortedByScore().filter((g) =>
    (g.builders ?? []).some((b) => matchesSlug(b, slug))
  )
  const solos = getSolosSortedByScore().filter((g) =>
    (g.builders ?? []).some((b) => matchesSlug(b, slug))
  )
  const blueprints = getBlueprintsSortedByScore().filter(
    (bp) => bp.builderName && builderSlug(bp.builderName) === slug
  )

  if (!guilds.length && !solos.length && !blueprints.length) return null

  // Derive canonical display name from found data
  let name: string | undefined
  for (const g of guilds) {
    for (const b of g.builders ?? []) {
      const n = formatBuilderName(b)
      if (n && builderSlug(n) === slug) { name = n; break }
    }
    if (name) break
  }
  if (!name) {
    for (const g of solos) {
      for (const b of g.builders ?? []) {
        const n = formatBuilderName(b)
        if (n && builderSlug(n) === slug) { name = n; break }
      }
      if (name) break
    }
  }
  if (!name && blueprints[0]?.builderName) name = blueprints[0].builderName

  return { name: name ?? slug, slug, guilds, solos, blueprints }
}
