import type { RankedGuild } from "@/types/guild"

export interface ScoringConfig {
  starScore: number
  likeScore: number
  loreBonus: number
  visitBonus: number
  weight4Threshold: number
  criticThreshold: number
  weight2Threshold: number
  weight1Threshold: number
  // Configurable weight value per tier (tier 0 = below weight1Threshold)
  w0: number
  w1: number
  w2: number
  w3: number
  w4: number
}

export const SCORING_DEFAULTS: ScoringConfig = {
  starScore: 2,
  likeScore: 1,
  loreBonus: 1,
  visitBonus: 1,
  weight4Threshold: 20,
  criticThreshold: 12,
  weight2Threshold: 8,
  weight1Threshold: 4,
  w0: 0,
  w1: 1,
  w2: 2,
  w3: 3,
  w4: 4,
}

const STAR = "⭐"
const THUMBS_EMOJIS = new Set(["👍", "👍🏻", "👍🏼", "👍🏽", "👍🏾", "👍🏿"])
const LIKE_EMOJIS = new Set(["🔥", "❤️"])

export function getVoterTier(threads: number, cfg: ScoringConfig): number {
  if (threads >= cfg.weight4Threshold) { return 4 }
  if (threads >= cfg.criticThreshold) { return 3 }
  if (threads >= cfg.weight2Threshold) { return 2 }
  if (threads >= cfg.weight1Threshold) { return 1 }
  return 0
}

const TIER_WEIGHTS: Array<keyof ScoringConfig> = ["w0", "w1", "w2", "w3", "w4"]

export function getVoterWeight(threads: number, cfg: ScoringConfig): number {
  return cfg[TIER_WEIGHTS[getVoterTier(threads, cfg)]] as number
}

// rawCaps optionally limits per-voter raw (pre-weight) points for this guild.
export function computeDynScore(
  g: RankedGuild,
  emojiMap: Record<string, string[]> | undefined,
  weights: Map<string, number>,
  cfg: ScoringConfig,
  blacklisted: Set<string>,
  rawCaps?: Map<string, number>,
): number {
  // Accumulate raw pts per voter first so caps and weights can be applied cleanly.
  const userRaw = new Map<string, number>()
  const thumbsVoters = new Set<string>()
  for (const [emoji, voters] of Object.entries(emojiMap ?? {})) {
    for (const v of voters) {
      if (blacklisted.has(v)) { continue }
      if (emoji === STAR) {
        userRaw.set(v, (userRaw.get(v) ?? 0) + cfg.starScore)
      } else if (THUMBS_EMOJIS.has(emoji)) {
        thumbsVoters.add(v)
      } else if (LIKE_EMOJIS.has(emoji)) {
        userRaw.set(v, (userRaw.get(v) ?? 0) + cfg.likeScore)
      }
    }
  }
  for (const v of thumbsVoters) {
    userRaw.set(v, (userRaw.get(v) ?? 0) + cfg.likeScore)
  }

  let total = 0
  for (const [v, raw] of userRaw) {
    const cap = rawCaps?.get(v)
    total += (cap !== undefined ? Math.min(raw, cap) : raw) * (weights.get(v) ?? 0)
  }
  let score = Math.round(total)
  if (g.lore) { score += cfg.loreBonus }
  if (g.whatToVisit) { score += cfg.visitBonus }
  return score
}

// tier is 0–4 (from getVoterTier); weight is the actual configured value.
export function weightColor(tier: number): string {
  if (tier >= 4) { return "bg-purple-500/20 text-purple-300 ring-1 ring-inset ring-purple-500/40" }
  if (tier >= 3) { return "bg-amber-500/20 text-amber-300 ring-1 ring-inset ring-amber-500/40" }
  if (tier >= 2) { return "bg-sky-500/20 text-sky-300 ring-1 ring-inset ring-sky-500/40" }
  if (tier >= 1) { return "bg-emerald-500/20 text-emerald-300 ring-1 ring-inset ring-emerald-500/40" }
  return "bg-muted/30 text-muted-foreground/40 ring-1 ring-inset ring-border/30"
}

export function weightLabel(tier: number, weight: number): string {
  const fmt = Number.isInteger(weight) ? String(weight) : weight.toFixed(1)
  if (tier === 3) { return `Critic ×${fmt}` }
  return `×${fmt}`
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
