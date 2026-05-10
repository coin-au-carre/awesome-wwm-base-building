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
