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
    badge: "bg-zinc-400/10 text-zinc-500 dark:text-zinc-400 ring-1 ring-inset ring-zinc-400/25",
  }
  const pct = rank / total
  if (pct <= 0.10) return {
    label: "Master",
    dot: "bg-fuchsia-500",
    badge: "text-fuchsia-700 dark:text-fuchsia-200 ring-2 ring-inset ring-fuchsia-400/60 font-bold",
    badgeStyle: { background: "linear-gradient(135deg, rgba(168,85,247,0.22) 0%, rgba(236,72,153,0.22) 100%)", boxShadow: "0 0 10px rgba(168,85,247,0.25)" },
  }
  if (pct <= 0.25) return {
    label: "Diamond",
    dot: "bg-sky-400",
    badge: "text-sky-700 dark:text-sky-200 ring-2 ring-inset ring-sky-400/50 font-semibold",
    badgeStyle: { background: "linear-gradient(135deg, rgba(34,211,238,0.18) 0%, rgba(99,102,241,0.18) 100%)", boxShadow: "0 0 8px rgba(34,211,238,0.2)" },
  }
  if (pct <= 0.40) return {
    label: "Platinum",
    dot: "bg-slate-300",
    badge: "text-slate-600 dark:text-slate-200 ring-1 ring-inset ring-slate-300/60 font-medium",
    badgeStyle: { background: "linear-gradient(135deg, rgba(203,213,225,0.18) 0%, rgba(148,163,184,0.12) 100%)", boxShadow: "0 0 6px rgba(203,213,225,0.2)" },
  }
  return {
    label: "Gold",
    dot: "bg-amber-400",
    badge: "bg-amber-400/15 text-amber-700 dark:text-amber-300 ring-1 ring-inset ring-amber-400/30",
  }
}

/** Strip Discord mention format: `<@773253240498159636> (GaoQingYang)` → `GaoQingYang`. */
export function formatBuilderName(raw: string): string {
  const match = raw.match(/<@\d+>\s*\((.+?)\)/)
  if (match) return match[1].trim()
  if (/^<@\d+>$/.test(raw.trim())) return ""
  return raw.trim()
}
