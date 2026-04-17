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

export type Tier = { label: string; dot: string; badge: string; badgeStyle?: Record<string, string> }

export function getTier(rank: number, total: number, score: number): Tier {
  if (score < 8) return {
    label: "Silver",
    dot: "bg-zinc-400",
    badge: "text-zinc-500 dark:text-zinc-400 ring-1 ring-inset ring-zinc-400/30",
    badgeStyle: { background: "rgba(113,113,122,0.08)" },
  }
  const pct = rank / total
  if (pct <= 0.10) return {
    label: "Master",
    dot: "bg-amber-300",
    badge: "tier-master text-amber-600 dark:text-amber-200 font-bold",
    badgeStyle: { background: "linear-gradient(135deg, rgba(251,191,36,0.18) 0%, rgba(245,158,11,0.12) 100%)" },
  }
  if (pct <= 0.25) return {
    label: "Diamond",
    dot: "bg-sky-300",
    badge: "text-sky-600 dark:text-sky-200 ring-1 ring-inset ring-sky-400/40 font-semibold",
    badgeStyle: { background: "linear-gradient(135deg, rgba(56,189,248,0.14) 0%, rgba(99,102,241,0.12) 100%)" },
  }
  if (pct <= 0.40) return {
    label: "Platinum",
    dot: "bg-cyan-200",
    badge: "text-cyan-700 dark:text-cyan-100 ring-1 ring-inset ring-cyan-200/50 font-medium",
    badgeStyle: { background: "linear-gradient(135deg, rgba(207,250,254,0.18) 0%, rgba(186,230,253,0.12) 100%)" },
  }
  return {
    label: "Gold",
    dot: "bg-amber-400",
    badge: "text-amber-700 dark:text-amber-300 ring-1 ring-inset ring-amber-400/35",
    badgeStyle: { background: "linear-gradient(135deg, rgba(251,191,36,0.14) 0%, rgba(245,158,11,0.08) 100%)" },
  }
}

/** Strip Discord mention format: `<@773253240498159636> (GaoQingYang)` → `GaoQingYang`. */
export function formatBuilderName(raw: string): string {
  const match = raw.match(/<@\d+>\s*\((.+?)\)/)
  if (match) return match[1].trim()
  if (/^<@\d+>$/.test(raw.trim())) return ""
  return raw.trim()
}
