import type { RankedGuild } from "@/types/guild"
import type { RankedBlueprint } from "@/types/blueprint"
import { getGuildsSortedByScore, getSolosSortedByScore } from "@/lib/guilds"
import { getBlueprintsSortedByScore } from "@/lib/blueprints"
import { formatBuilderName, builderSlug } from "@/lib/format"
import { resolveCanonical, getAllSlugsForCanonical } from "@/lib/builder-aliases"
import { HOMESTEAD_SHEETS, type HomesteadSheet } from "@/lib/homestead-resources"

export interface BuilderProfile {
  name: string
  slug: string
  guilds: RankedGuild[]
  solos: RankedGuild[]
  blueprints: RankedBlueprint[]
  homesteadSheets: HomesteadSheet[]
}

function matchesSlug(rawName: string, targetCanonical: string): boolean {
  const cleaned = formatBuilderName(rawName)
  if (!cleaned) return false
  return resolveCanonical(builderSlug(cleaned)) === targetCanonical
}

export function getAllBuilderSlugs(): { name: string; slug: string }[] {
  const slugMap = new Map<string, string>() // canonical slug → display name

  function addName(name: string) {
    const s = builderSlug(name)
    if (!s) return
    const canonical = resolveCanonical(s)
    if (!slugMap.has(canonical)) slugMap.set(canonical, name)
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
  for (const sheet of HOMESTEAD_SHEETS) {
    addName(sheet.by)
  }

  return [...slugMap.entries()].map(([slug, name]) => ({ name, slug }))
}

export interface BuilderRosterEntry {
  name: string
  slug: string
  coverImage?: string
  guildCount: number
  soloCount: number
  blueprintCount: number
  homesteadSheetCount: number
  tutorialCount: number
}

// Every builder credited on at least one guild base, solo build,
// blueprint, homestead sheet, or tutorial — no minimum-diversity filter
// (unlike getNotableBuilders below, which curates a smaller "wall of
// fame"). Used by the /builders directory, which wants everyone.
export function getBuilderRoster(tutorialCountBySlug?: Map<string, number>): BuilderRosterEntry[] {
  const bySlug = new Map<string, BuilderRosterEntry>()

  function get(slug: string, name: string): BuilderRosterEntry {
    if (!bySlug.has(slug)) bySlug.set(slug, { name, slug, guildCount: 0, soloCount: 0, blueprintCount: 0, homesteadSheetCount: 0, tutorialCount: 0 })
    return bySlug.get(slug)!
  }

  for (const g of getGuildsSortedByScore()) {
    for (const b of g.builders ?? []) {
      const name = formatBuilderName(b)
      if (!name) continue
      const s = resolveCanonical(builderSlug(name))
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
      const s = resolveCanonical(builderSlug(name))
      if (!s) continue
      const entry = get(s, name)
      if (!entry.coverImage) entry.coverImage = g.coverImage ?? g.screenshots?.[0]
      entry.soloCount++
    }
  }

  for (const bp of getBlueprintsSortedByScore()) {
    if (!bp.builderName) continue
    const s = resolveCanonical(builderSlug(bp.builderName))
    if (!s) continue
    const entry = get(s, bp.builderName)
    if (!entry.coverImage) entry.coverImage = bp.coverImage ?? bp.screenshots?.[0]
    entry.blueprintCount++
  }

  for (const sheet of HOMESTEAD_SHEETS) {
    const s = resolveCanonical(builderSlug(sheet.by))
    if (!s) continue
    get(s, sheet.by).homesteadSheetCount++
  }

  if (tutorialCountBySlug) {
    for (const [s, count] of tutorialCountBySlug) {
      if (bySlug.has(s)) bySlug.get(s)!.tutorialCount = count
    }
  }

  return [...bySlug.values()]
}

export interface NotableBuilder extends BuilderRosterEntry {
  typeCount: number
}

export function getNotableBuilders(
  tutorialCountBySlug?: Map<string, number>,
  limit?: number,
): NotableBuilder[] {
  // typeCount deliberately excludes homesteadSheetCount, matching this
  // function's original curation criteria — adding it here would newly
  // qualify builders on the credits.astro "notable builders" wall who
  // previously didn't meet the bar, an unrelated behavior change.
  return getBuilderRoster(tutorialCountBySlug)
    .map((e) => ({ ...e, typeCount: [e.guildCount, e.soloCount, e.blueprintCount, e.tutorialCount].filter(Boolean).length }))
    .filter((e) => e.typeCount >= 2)
    .sort((a, b) => {
      // 1. tutorial authors first
      if (b.tutorialCount !== a.tutorialCount) return b.tutorialCount - a.tutorialCount
      // 2. most diverse contributors next
      if (b.typeCount !== a.typeCount) return b.typeCount - a.typeCount
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
      const s = resolveCanonical(builderSlug(formatBuilderName(b)))
      if (s && !seenGuild.has(s)) { seenGuild.add(s); add(s) }
    }
  }

  const seenSolo = new Set<string>()
  for (const g of getSolosSortedByScore()) {
    for (const b of g.builders ?? []) {
      const s = resolveCanonical(builderSlug(formatBuilderName(b)))
      if (s && !seenSolo.has(s)) { seenSolo.add(s); add(s) }
    }
  }

  const seenBlueprint = new Set<string>()
  for (const bp of getBlueprintsSortedByScore()) {
    if (!bp.builderName) continue
    const s = resolveCanonical(builderSlug(bp.builderName))
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
  const canonical = resolveCanonical(slug)
  const allSlugs = getAllSlugsForCanonical(canonical)

  const guilds = getGuildsSortedByScore().filter((g) =>
    (g.builders ?? []).some((b) => matchesSlug(b, canonical))
  )
  const solos = getSolosSortedByScore().filter((g) =>
    (g.builders ?? []).some((b) => matchesSlug(b, canonical))
  )
  const blueprints = getBlueprintsSortedByScore().filter(
    (bp) => bp.builderName && allSlugs.has(resolveCanonical(builderSlug(bp.builderName)))
  )
  const homesteadSheets = HOMESTEAD_SHEETS.filter((sheet) => matchesSlug(sheet.by, canonical))

  if (!guilds.length && !solos.length && !blueprints.length && !homesteadSheets.length) return null

  // Prefer a name whose slug IS the canonical (not an alias form)
  let name: string | undefined
  let aliasFormName: string | undefined
  for (const g of [...guilds, ...solos]) {
    for (const b of g.builders ?? []) {
      const n = formatBuilderName(b)
      if (!n || resolveCanonical(builderSlug(n)) !== canonical) continue
      if (builderSlug(n) === canonical) { name = n; break }
      if (!aliasFormName) aliasFormName = n
    }
    if (name) break
  }
  if (!name && blueprints[0]?.builderName) name = blueprints[0].builderName
  if (!name && homesteadSheets[0]) name = homesteadSheets[0].by
  if (!name) name = aliasFormName

  return { name: name ?? canonical, slug: canonical, guilds, solos, blueprints, homesteadSheets }
}
