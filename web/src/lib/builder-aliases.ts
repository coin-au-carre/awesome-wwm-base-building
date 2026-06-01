// Maps alias slug → canonical slug.
// Add a line here whenever two differently-spelled names refer to the same builder.
// Keys and values are slugs (output of slugify/builderSlug), not display names.
export const BUILDER_ALIASES: Record<string, string> = {
  "diana念": "ðìana",
  "siming-司命": "siming司命",
  "aegisnite-edge": "aegisnite",
  "ℭ𝔞𝔯𝔫𝔦": "carnii",
  "kira": "kirakosma",
}

export function resolveCanonical(slug: string): string {
  return BUILDER_ALIASES[slug] ?? slug
}

export function getAllSlugsForCanonical(canonicalSlug: string): Set<string> {
  const set = new Set([canonicalSlug])
  for (const [alias, canonical] of Object.entries(BUILDER_ALIASES)) {
    if (canonical === canonicalSlug) { set.add(alias) }
  }
  return set
}
