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

export type Tier = { label: string; dot: string; badge: string; badgeStyle?: Record<string, string> }

// Tier thresholds — edit these to adjust how many guilds fall into each tier.
// Values are percentages of total guilds (0.10 = top 10%).
// Silver is score-based, not rank-based.
export const TIER_THRESHOLDS = {
  master: 0.15,
  diamond: 0.30,
  platinum: 0.50,
                     // rest    → Gold

  silverMaxScore: 1, // below this score → Silver
}

const RANK_MEDALS: Record<number, string> = { 1: "🥇", 2: "🥈", 3: "🥉" }
export function rankLabel(rank: number): string {
  return RANK_MEDALS[rank] ?? String(rank)
}

export function getTier(rank: number, total: number, score: number): Tier {
  if (score < TIER_THRESHOLDS.silverMaxScore) {
    return {
      label: "Silver",
      dot: "bg-zinc-400",
      badge: "text-zinc-500 dark:text-zinc-400 ring-1 ring-inset ring-zinc-400/30",
      badgeStyle: { background: "rgba(113,113,122,0.08)" },
    }
  }
  const pct = rank / total
  if (pct <= TIER_THRESHOLDS.master) {
    return {
      label: "Master",
      dot: "bg-amber-300",
      badge: "tier-master text-amber-700 dark:text-amber-200 font-bold ring-1 ring-inset ring-amber-400/50",
      badgeStyle: { background: "linear-gradient(135deg, rgba(251,191,36,0.32) 0%, rgba(245,158,11,0.22) 100%)" },
    }
  }
  if (pct <= TIER_THRESHOLDS.diamond) {
    return {
      label: "Diamond",
      dot: "bg-sky-300",
      badge: "text-sky-600 dark:text-sky-200 ring-1 ring-inset ring-sky-400/40 font-semibold",
      badgeStyle: { background: "linear-gradient(135deg, rgba(56,189,248,0.14) 0%, rgba(99,102,241,0.12) 100%)" },
    }
  }
  if (pct <= TIER_THRESHOLDS.platinum) {
    return {
      label: "Platinum",
      dot: "bg-cyan-200",
      badge: "text-cyan-700 dark:text-cyan-100 ring-1 ring-inset ring-cyan-200/50 font-medium",
      badgeStyle: { background: "linear-gradient(135deg, rgba(207,250,254,0.18) 0%, rgba(186,230,253,0.12) 100%)" },
    }
  }
  return {
    label: "Gold",
    dot: "bg-amber-400",
    badge: "text-amber-700 dark:text-amber-300 ring-1 ring-inset ring-amber-400/35",
    badgeStyle: { background: "linear-gradient(135deg, rgba(251,191,36,0.14) 0%, rgba(245,158,11,0.08) 100%)" },
  }
}
