import { readFileSync } from "fs"
import { resolve } from "path"
import type { Interior, IndexedInterior } from "@/types/interior"
import { slugify } from "@/lib/format"
import { parseLastModified } from "@/lib/dates"

function repoFile(relativePath: string) {
  return resolve(process.cwd(), "..", relativePath)
}

function loadJSON(): Interior[] {
  try {
    const raw = readFileSync(repoFile("data/interior.json"), "utf-8")
    return JSON.parse(raw)
  } catch {
    return []
  }
}

function sortByDate(items: Interior[]): IndexedInterior[] {
  return [...items]
    .sort((a, b) => parseLastModified(b.createdAt) - parseLastModified(a.createdAt))
    .map((it) => ({ ...it, slug: slugify(it.name) }))
}

const ALL_INTERIORS: Interior[] = loadJSON()
const SORTED_INTERIORS = sortByDate(ALL_INTERIORS)

export function getInteriorsSortedByDate(): IndexedInterior[] { return SORTED_INTERIORS }
export function hasInteriors(): boolean { return ALL_INTERIORS.length > 0 }
