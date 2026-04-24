import { useState, useMemo } from "react"
import type { RankedGuild } from "@/types/guild"
import type { ReactionMap, UserMap } from "@/lib/guilds"
import { formatBuilderName, stripGuildShowcase } from "@/lib/slugify"
import { url } from "@/lib/url"
import { cn } from "@/lib/utils"
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"

interface ScoringConfig {
  starScore: number
  likeScore: number
  loreBonus: number
  visitBonus: number
  criticThreshold: number
  weight2Threshold: number
  weight1Threshold: number
}

const DEFAULTS: ScoringConfig = {
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

const PAGE_SIZE = 50

function threadID(g: RankedGuild): string {
  return g.discordThread.split("/").at(-1) ?? ""
}

function displayName(userID: string, users: UserMap): string {
  const u = users[userID]
  if (!u) { return userID }
  return u.nickname || u.globalName || u.username
}

function getVoterWeight(threads: number, cfg: ScoringConfig): number {
  if (threads >= cfg.criticThreshold) { return 3 }
  if (threads >= cfg.weight2Threshold) { return 2 }
  if (threads >= cfg.weight1Threshold) { return 1 }
  return 0
}

function computeDynScore(
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

function weightColor(w: number): string {
  if (w >= 3) { return "bg-amber-500/20 text-amber-300 ring-1 ring-inset ring-amber-500/40" }
  if (w >= 2) { return "bg-sky-500/20 text-sky-300 ring-1 ring-inset ring-sky-500/40" }
  if (w >= 1) { return "bg-emerald-500/20 text-emerald-300 ring-1 ring-inset ring-emerald-500/40" }
  return "bg-muted/30 text-muted-foreground/40 ring-1 ring-inset ring-border/30"
}

function weightLabel(w: number): string {
  if (w >= 3) { return "Critic ×3" }
  if (w >= 2) { return "×2" }
  if (w >= 1) { return "×1" }
  return "×0"
}

function NumInput({
  label,
  value,
  onChange,
}: {
  label: string
  value: number
  onChange: (v: number) => void
}) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[10px] text-muted-foreground whitespace-nowrap">{label}</span>
      <Input
        type="number"
        min={0}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="h-7 w-16 text-xs px-2"
      />
    </div>
  )
}

interface Props {
  guilds: RankedGuild[]
  reactions: ReactionMap
  users: UserMap
  voterBlacklist: string[]
}

export function AdminLeaderboard({ guilds, reactions, users, voterBlacklist }: Props) {
  const [cfg, setCfg] = useState<ScoringConfig>(DEFAULTS)
  const [filterVoter, setFilterVoter] = useState("")
  const [page, setPage] = useState(1)
  const [disabledBlacklist, setDisabledBlacklist] = useState<Set<string>>(new Set(voterBlacklist))

  function set(key: keyof ScoringConfig, value: number) {
    setCfg((prev) => ({ ...prev, [key]: value }))
    setPage(1)
  }

  const voterCounts = useMemo(() => {
    const counts = new Map<string, number>()
    for (const g of guilds) {
      const emojiMap = reactions[threadID(g)] ?? {}
      const seen = new Set<string>()
      for (const voters of Object.values(emojiMap)) {
        for (const v of voters) {
          if (!disabledBlacklist.has(v)) { seen.add(v) }
        }
      }
      for (const v of seen) { counts.set(v, (counts.get(v) ?? 0) + 1) }
    }
    return counts
  }, [guilds, reactions, disabledBlacklist])

  const weights = useMemo(() => {
    const map = new Map<string, number>()
    for (const [voter, count] of voterCounts) {
      map.set(voter, getVoterWeight(count, cfg))
    }
    return map
  }, [voterCounts, cfg])

  const ranked = useMemo(() => {
    const withScore = guilds.map((g) => ({
      ...g,
      dynScore: computeDynScore(g, reactions[threadID(g)], weights, cfg, disabledBlacklist),
    }))
    withScore.sort((a, b) => b.dynScore - a.dynScore)
    return withScore.reduce<Array<(typeof withScore)[0] & { dynRank: number }>>((acc, g, i) => {
      const dynRank = i === 0 ? 1 : (g.dynScore < withScore[i - 1].dynScore ? i + 1 : acc[i - 1].dynRank)
      acc.push({ ...g, dynRank })
      return acc
    }, [])
  }, [guilds, reactions, weights, cfg, disabledBlacklist])

  const filtered = useMemo(() => {
    if (!filterVoter.trim()) { return ranked }
    const q = filterVoter.toLowerCase()
    return ranked.filter((g) => {
      const emojiMap = reactions[threadID(g)] ?? {}
      return Object.values(emojiMap).some((voters) =>
        voters.some((v) => displayName(v, users).toLowerCase().includes(q))
      )
    })
  }, [ranked, filterVoter, reactions, users])

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE)
  const paginated = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  return (
    <div className="space-y-6">
      <div className="rounded-xl ring-1 ring-border bg-muted/20 p-4 space-y-4">
        <p className="text-sm font-semibold">Scoring Configuration</p>
        <div className="flex flex-wrap gap-8">
          <div className="space-y-2">
            <p className="text-[10px] font-medium uppercase tracking-widest text-muted-foreground">
              Emoji weights
            </p>
            <div className="flex gap-3 flex-wrap">
              <NumInput label="⭐ pts" value={cfg.starScore} onChange={(v) => set("starScore", v)} />
              <NumInput label="👍/🔥/❤️ pts" value={cfg.likeScore} onChange={(v) => set("likeScore", v)} />
              <NumInput label="Lore bonus" value={cfg.loreBonus} onChange={(v) => set("loreBonus", v)} />
              <NumInput label="Visit bonus" value={cfg.visitBonus} onChange={(v) => set("visitBonus", v)} />
            </div>
          </div>
          <div className="space-y-2">
            <p className="text-[10px] font-medium uppercase tracking-widest text-muted-foreground">
              Voter weight thresholds (distinct threads reacted to)
            </p>
            <div className="flex gap-3 flex-wrap">
              <NumInput
                label="Critic ×3 ≥"
                value={cfg.criticThreshold}
                onChange={(v) => set("criticThreshold", v)}
              />
              <NumInput
                label="×2 ≥"
                value={cfg.weight2Threshold}
                onChange={(v) => set("weight2Threshold", v)}
              />
              <NumInput
                label="×1 ≥"
                value={cfg.weight1Threshold}
                onChange={(v) => set("weight1Threshold", v)}
              />
            </div>
          </div>
        </div>
        <div className="flex gap-2 flex-wrap items-center">
          <span className="text-[10px] text-muted-foreground">Voter legend:</span>
          {[3, 2, 1, 0].map((w) => (
            <span
              key={w}
              className={cn(
                "inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium",
                weightColor(w)
              )}
            >
              {weightLabel(w)}
            </span>
          ))}
          <span className="text-[10px] text-muted-foreground ml-2">
            Hover any voter badge to see their thread count and weight.
          </span>
        </div>
      </div>

      {voterBlacklist.length > 0 && (
        <div className="rounded-xl ring-1 ring-border bg-muted/20 p-4 space-y-3">
          <p className="text-sm font-semibold">Blacklisted voters</p>
          <div className="flex flex-wrap gap-2">
            {voterBlacklist.map((uid) => {
              const excluded = disabledBlacklist.has(uid)
              const name = displayName(uid, users)
              return (
                <button
                  key={uid}
                  onClick={() => {
                    setDisabledBlacklist((prev) => {
                      const next = new Set(prev)
                      if (next.has(uid)) { next.delete(uid) }
                      else { next.add(uid) }
                      return next
                    })
                    setPage(1)
                  }}
                  title={uid}
                  className={cn(
                    "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ring-1 ring-inset transition-colors cursor-pointer",
                    excluded
                      ? "bg-rose-500/20 text-rose-300 ring-rose-500/40"
                      : "bg-emerald-500/20 text-emerald-300 ring-emerald-500/40"
                  )}
                >
                  <span>{excluded ? "✗" : "✓"}</span>
                  {name}
                </button>
              )
            })}
          </div>
          <p className="text-[10px] text-muted-foreground">
            Red = excluded from scoring (default). Click to include their votes and see the score difference.
          </p>
        </div>
      )}

      <div className="flex items-center gap-3 flex-wrap">
        <Input
          type="text"
          placeholder="Filter by voter name…"
          value={filterVoter}
          onChange={(e) => {
            setFilterVoter(e.target.value)
            setPage(1)
          }}
          className="h-8 w-64 text-sm"
        />
        <span className="text-xs text-muted-foreground">
          {filtered.length} / {guilds.length} guilds
        </span>
      </div>

      <div className="rounded-xl ring-1 ring-border overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-muted/40 text-muted-foreground">
              <TableHead className="w-14 text-center">#</TableHead>
              <TableHead>Guild</TableHead>
              <TableHead className="hidden md:table-cell">Builders</TableHead>
              <TableHead className="w-20 text-right">Score</TableHead>
              <TableHead className="w-14 text-center">Shots</TableHead>
              <TableHead>Reactions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {paginated.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="py-8 text-center text-muted-foreground">
                  No guilds match your filter.
                </TableCell>
              </TableRow>
            ) : (
              paginated.map((g) => {
                const emojiMap = reactions[threadID(g)] ?? {}
                return (
                  <TableRow key={g.name} className="align-top">
                    <TableCell className="text-center font-mono text-sm text-muted-foreground pt-3">
                      #{g.dynRank}
                    </TableCell>
                    <TableCell className="pt-3">
                      <div className="flex flex-col gap-0.5">
                        <a
                          href={url(`/guilds/${g.slug}`)}
                          className="font-medium text-sm hover:underline"
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          {stripGuildShowcase(g.guildName || g.name)}
                        </a>
                        {g.lore && (
                          <span
                            className="text-[10px] text-muted-foreground/60 cursor-default"
                            title={g.lore}
                          >
                            📖 lore
                          </span>
                        )}
                        {g.whatToVisit && (
                          <span
                            className="text-[10px] text-muted-foreground/60 cursor-default"
                            title={g.whatToVisit}
                          >
                            📍 visit
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="hidden md:table-cell text-sm text-muted-foreground pt-3 max-w-35">
                      <span
                        className="block truncate"
                        title={(g.builders ?? []).map(formatBuilderName).filter(Boolean).join(", ") || "—"}
                      >
                        {(g.builders ?? []).map(formatBuilderName).filter(Boolean).join(", ") || "—"}
                      </span>
                    </TableCell>
                    <TableCell className="text-right pt-3">
                      <div className="flex flex-col items-end">
                        <span className="font-mono font-semibold text-sm">{g.dynScore}</span>
                        {g.dynScore !== g.score && (
                          <span
                            className="text-[10px] text-muted-foreground/40 cursor-default"
                            title="Stored score (computed at snapshot time, without voter weights)"
                          >
                            was {g.score}
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-center text-sm text-muted-foreground pt-3">
                      {g.screenshots?.length ?? 0}
                    </TableCell>
                    <TableCell className="pt-2">
                      <div className="flex flex-wrap gap-2">
                        {Object.entries(emojiMap).map(([emoji, voters]) => {
                          if (!voters.length) { return null }
                          return (
                            <div key={emoji} className="flex items-start gap-1 flex-wrap">
                              <span className="text-sm leading-5 shrink-0">{emoji}</span>
                              <div className="flex flex-wrap gap-1">
                                {voters.map((voter) => {
                                  const w = weights.get(voter) ?? 0
                                  const name = displayName(voter, users)
                                  const highlight =
                                    filterVoter.trim() &&
                                    name.toLowerCase().includes(filterVoter.toLowerCase())
                                  return (
                                    <span
                                      key={voter}
                                      title={`${name} (${voter}) — ${voterCounts.get(voter) ?? 0} distinct threads, weight ×${w}`}
                                      className={cn(
                                        "inline-flex items-center rounded-full px-1.5 py-0.5 text-[10px] font-medium cursor-default",
                                        weightColor(w),
                                        highlight && "ring-2 ring-primary"
                                      )}
                                    >
                                      {name}
                                    </span>
                                  )
                                })}
                              </div>
                            </div>
                          )
                        })}
                        {!Object.keys(emojiMap).length && (
                          <span className="text-xs text-muted-foreground/30">—</span>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center gap-1">
          <Button
            variant="outline"
            size="xs"
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page === 1}
          >
            ←
          </Button>
          <span className="text-xs px-2 text-muted-foreground">
            {page} / {totalPages}
          </span>
          <Button
            variant="outline"
            size="xs"
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
          >
            →
          </Button>
        </div>
      )}
    </div>
  )
}
