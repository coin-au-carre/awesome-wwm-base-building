// Maps alias slug → canonical slug.
// Add a line here whenever two differently-spelled names refer to the same builder.
export const BUILDER_ALIASES: Record<string, string> = {
  "diana念": "ðìana",
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
