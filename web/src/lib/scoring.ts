import type { RankedGuild } from "@/types/guild"

export interface ScoringConfig {
  starScore: number
  likeScore: number
  loreBonus: number
  visitBonus: number
  criticThreshold: number
  weight2Threshold: number
  weight1Threshold: number
}

export const SCORING_DEFAULTS: ScoringConfig = {
  starScore: 2,
  likeScore: 1,
  loreBonus: 1,
  visitBonus: 1,
  criticThreshold: 12,
  weight2Threshold: 8,
  weight1Threshold: 4,
}

const STAR = "⭐"
const THUMBS_EMOJIS = new Set(["👍", "👍🏻", "👍🏼", "👍🏽", "👍🏾", "👍🏿"])
const LIKE_EMOJIS = new Set(["🔥", "❤️"])

export function getVoterWeight(threads: number, cfg: ScoringConfig): number {
  if (threads >= cfg.criticThreshold) { return 3 }
  if (threads >= cfg.weight2Threshold) { return 2 }
  if (threads >= cfg.weight1Threshold) { return 1 }
  return 0
}

export function computeDynScore(
  g: RankedGuild,
  emojiMap: Record<string, string[]> | undefined,
  weights: Map<string, number>,
  cfg: ScoringConfig,
  blacklisted: Set<string>,
): number {
  let score = 0
  // Deduplicate thumbs-up voters across all skin-tone variants.
  const thumbsVoters = new Set<string>()
  for (const [emoji, voters] of Object.entries(emojiMap ?? {})) {
    if (emoji === STAR) {
      for (const v of voters) {
        if (!blacklisted.has(v)) { score += cfg.starScore * (weights.get(v) ?? 0) }
      }
    } else if (THUMBS_EMOJIS.has(emoji)) {
      for (const v of voters) {
        if (!blacklisted.has(v)) { thumbsVoters.add(v) }
      }
    } else if (LIKE_EMOJIS.has(emoji)) {
      for (const v of voters) {
        if (!blacklisted.has(v)) { score += cfg.likeScore * (weights.get(v) ?? 0) }
      }
    }
  }
  for (const v of thumbsVoters) {
    score += cfg.likeScore * (weights.get(v) ?? 0)
  }
  if (g.lore) { score += cfg.loreBonus }
  if (g.whatToVisit) { score += cfg.visitBonus }
  return score
}

export function weightColor(w: number): string {
  if (w >= 3) { return "bg-amber-500/20 text-amber-300 ring-1 ring-inset ring-amber-500/40" }
  if (w >= 2) { return "bg-sky-500/20 text-sky-300 ring-1 ring-inset ring-sky-500/40" }
  if (w >= 1) { return "bg-emerald-500/20 text-emerald-300 ring-1 ring-inset ring-emerald-500/40" }
  return "bg-muted/30 text-muted-foreground/40 ring-1 ring-inset ring-border/30"
}

export function weightLabel(w: number): string {
  if (w >= 3) { return "Critic ×3" }
  if (w >= 2) { return "×2" }
  if (w >= 1) { return "×1" }
  return "×0"
}
