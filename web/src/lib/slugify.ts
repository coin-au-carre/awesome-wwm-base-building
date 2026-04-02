/**
 * Port of Go's Slugify() from internal/generator/page.go.
 * Must produce identical output for all guild names.
 */
export function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, "-")
    .replace(/^-+|-+$/g, "")
}

const RANK_MEDALS: Record<number, string> = { 1: "🥇", 2: "🥈", 3: "🥉" }
export function rankLabel(rank: number): string {
  return RANK_MEDALS[rank] ?? String(rank)
}
