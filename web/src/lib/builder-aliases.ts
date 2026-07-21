import identities from "../../../data/builder_identities.json"
import { slugify } from "@/lib/format"

// See docs/builder-identity.md — this file's data now lives in
// data/builder_identities.json (one record per real builder), populated by
// the /wwm-uid Discord command going forward. canonicalSlug is always
// slugify(canonicalAlias) — lowercase, the public URL/matching identity.
// canonicalAlias and aliases keep their natural display casing; they're
// slugified here at load time rather than needing to be pre-slugified by
// hand (the old inline BUILDER_ALIASES map required that, which is how one
// entry — "Kuri (SiMing 司命)" — ended up never actually matching anything).
export interface BuilderIdentity {
  discordId?: string
  canonicalAlias: string
  canonicalSlug: string
  aliases?: string[]
  ingameNickname?: string
  neteaseNumberId?: string
  neteasePid?: string
  neteaseHostnum?: number
}

// The full parsed list, re-exported so any caller (including .astro
// frontmatter — e.g. builders/[slug].astro's getStaticPaths, gallery.astro,
// gallery/builder.astro) can read it via a real module import rather than
// each hand-rolling its own readFileSync. That matters more than it might
// look: Astro's build extracts getStaticPaths into its own isolated
// compiled chunk, separate from the rest of a component's frontmatter
// scope — a same-file helper function defined alongside it does *not*
// reliably survive that extraction (confirmed the hard way: "X is not
// defined" at build time), but a genuine cross-module import does.
export const BUILDER_IDENTITIES = identities as BuilderIdentity[]

// Alias slug → canonical slug, built once from every record's aliases —
// replaces the old inline BUILDER_ALIASES object literal. A plain JSON
// import (not readFileSync) on purpose: this module is also bundled for
// the browser (imported by client-hydrated LeaderboardTable/BlueprintGrid/
// TutorialsFilter), where Node's fs/path don't exist — Vite inlines JSON
// imports as plain data at build time, safe in both server and client
// bundles, unlike guilds.ts/nav-versions.ts's readFileSync pattern, which
// only works because those are only ever imported from .astro frontmatter.
const ALIAS_TO_CANONICAL: Record<string, string> = {}
for (const entry of BUILDER_IDENTITIES) {
  for (const alias of entry.aliases ?? []) {
    ALIAS_TO_CANONICAL[slugify(alias)] = entry.canonicalSlug
  }
}

export function resolveCanonical(slug: string): string {
  return ALIAS_TO_CANONICAL[slug] ?? slug
}

export function getAllSlugsForCanonical(canonicalSlug: string): Set<string> {
  const set = new Set([canonicalSlug])
  for (const [alias, canonical] of Object.entries(ALIAS_TO_CANONICAL)) {
    if (canonical === canonicalSlug) { set.add(alias) }
  }
  return set
}
