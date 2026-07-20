import { readFileSync } from "fs"
import { resolve } from "path"
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

function repoFile(p: string) {
  return resolve(process.cwd(), "..", p)
}

function loadIdentities(): BuilderIdentity[] {
  try {
    return JSON.parse(readFileSync(repoFile("data/builder_identities.json"), "utf-8"))
  } catch {
    return []
  }
}

// Alias slug → canonical slug, built once from every record's aliases —
// replaces the old inline BUILDER_ALIASES object literal.
const ALIAS_TO_CANONICAL: Record<string, string> = {}
for (const entry of loadIdentities()) {
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
