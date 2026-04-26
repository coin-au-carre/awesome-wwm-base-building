import { useState, useMemo } from "react"
import type { RankedGuild } from "@/types/guild"
import type { ReactionMap, UserMap } from "@/lib/guilds"
import { formatBuilderName, stripGuildShowcase } from "@/lib/format"
import { url } from "@/lib/url"
import { cn } from "@/lib/utils"
import { type ScoringConfig, SCORING_DEFAULTS, getVoterWeight, computeDynScore, weightColor, weightLabel } from "@/lib/scoring"
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

const PAGE_SIZE = 50
const VOTER_PAGE_SIZE = 30

function threadID(g: RankedGuild): string {
  return g.discordThread.split("/").at(-1) ?? ""
}

function displayName(userID: string, users: UserMap): string {
  const u = users[userID]
  if (!u) { return userID }
  return u.nickname || u.globalName || u.username
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
  const [cfg, setCfg] = useState<ScoringConfig>(SCORING_DEFAULTS)
  const [filterVoter, setFilterVoter] = useState("")
  const [page, setPage] = useState(1)
  const [voterPage, setVoterPage] = useState(1)
  const [disabledBlacklist, setDisabledBlacklist] = useState<Set<string>>(new Set(voterBlacklist))
  const [copiedUid, setCopiedUid] = useState(false)
  const [selectedVoterUid, setSelectedVoterUid] = useState<string | null>(null)
  const [enableAbuseCap, setEnableAbuseCap] = useState(true)
  const [votersOpen, setVotersOpen] = useState(false)

  function copyUid(uid: string) {
    navigator.clipboard.writeText(uid)
    setCopiedUid(true)
    setTimeout(() => setCopiedUid(false), 1500)
  }

  function filterByVoter(uid: string) {
    setFilterVoter(displayName(uid, users))
    setSelectedVoterUid(uid)
    setPage(1)
  }

  function set(key: keyof ScoringConfig, value: number) {
    setCfg((prev) => ({ ...prev, [key]: value }))
    setPage(1)
  }

  // Mirrors the Go abuse detection constants.
  const ABUSE_MIN_THREADS = 4
  const ABUSE_HIGH_SCORE = 4    // raw pts threshold (⭐ + at least 2 others)
  const ABUSE_MIN_HIGH_OTHERS = 1

  // threadID → userID → raw pt cap for that voter on that guild.
  const abuseCaps = useMemo(() => {
    const THUMBS = new Set(["👍", "👍🏻", "👍🏼", "👍🏽", "👍🏾", "👍🏿"])
    const userThreadPts = new Map<string, Map<string, number>>()
    for (const g of guilds) {
      const tid = threadID(g)
      const emojiMap = reactions[tid] ?? {}
      const userRaw = new Map<string, number>()
      const thumbsVoters = new Set<string>()
      for (const [emoji, voters] of Object.entries(emojiMap)) {
        for (const v of voters) {
          if (disabledBlacklist.has(v)) { continue }
          if (emoji === "⭐") { userRaw.set(v, (userRaw.get(v) ?? 0) + 2) }
          else if (THUMBS.has(emoji)) { thumbsVoters.add(v) }
          else if (emoji === "🔥" || emoji === "❤️") { userRaw.set(v, (userRaw.get(v) ?? 0) + 1) }
        }
      }
      for (const v of thumbsVoters) { userRaw.set(v, (userRaw.get(v) ?? 0) + 1) }
      for (const [uid, pts] of userRaw) {
        if (!userThreadPts.has(uid)) { userThreadPts.set(uid, new Map()) }
        userThreadPts.get(uid)!.set(tid, pts)
      }
    }

    const result = new Map<string, Map<string, number>>()
    for (const [uid, byThread] of userThreadPts) {
      if (byThread.size < ABUSE_MIN_THREADS) { continue }
      let topPts = 0, topTid = "", total = 0
      for (const [tid, pts] of byThread) {
        total += pts
        if (pts > topPts) { topPts = pts; topTid = tid }
      }
      if (topPts < ABUSE_HIGH_SCORE) { continue }
      let highOthers = 0
      for (const [tid, pts] of byThread) {
        if (tid !== topTid && pts >= ABUSE_HIGH_SCORE) { highOthers++ }
      }
      if (highOthers >= ABUSE_MIN_HIGH_OTHERS) { continue }
      const cap = Math.ceil((total - topPts) / (byThread.size - 1))
      if (!result.has(topTid)) { result.set(topTid, new Map()) }
      result.get(topTid)!.set(uid, cap)
    }
    return result
  }, [guilds, reactions, disabledBlacklist])

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

  const totalReactions = useMemo(() => {
    const counts = new Map<string, number>()
    for (const g of guilds) {
      const emojiMap = reactions[threadID(g)] ?? {}
      for (const voters of Object.values(emojiMap)) {
        for (const v of voters) {
          if (!disabledBlacklist.has(v)) { counts.set(v, (counts.get(v) ?? 0) + 1) }
        }
      }
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

  const voterLeaderboard = useMemo(() => {
    const all = Array.from(voterCounts.entries()).map(([uid, threads]) => ({
      uid,
      threads,
      reactions: totalReactions.get(uid) ?? 0,
      weight: weights.get(uid) ?? 0,
    }))
    all.sort((a, b) => b.threads - a.threads || b.reactions - a.reactions)
    return all
  }, [voterCounts, totalReactions, weights])

  const ranked = useMemo(() => {
    const withScore = guilds.map((g) => {
      const tid = threadID(g)
      const caps = enableAbuseCap ? abuseCaps.get(tid) : undefined
      return {
        ...g,
        dynScore: computeDynScore(g, reactions[tid], weights, cfg, disabledBlacklist, caps),
      }
    })
    withScore.sort((a, b) => b.dynScore - a.dynScore)
    return withScore.reduce<Array<(typeof withScore)[0] & { dynRank: number }>>((acc, g, i) => {
      const dynRank = i === 0 ? 1 : (g.dynScore < withScore[i - 1].dynScore ? i + 1 : acc[i - 1].dynRank)
      acc.push({ ...g, dynRank })
      return acc
    }, [])
  }, [guilds, reactions, weights, cfg, disabledBlacklist, abuseCaps, enableAbuseCap])

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
      <div className="sticky top-0 z-10 rounded-xl ring-1 ring-border bg-background/95 backdrop-blur p-4 space-y-4">
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
        <div className="flex items-center gap-3">
          <button
            onClick={() => { setEnableAbuseCap((v) => !v); setPage(1) }}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ring-1 ring-inset transition-colors cursor-pointer",
              enableAbuseCap
                ? "bg-amber-500/20 text-amber-300 ring-amber-500/40"
                : "bg-muted/30 text-muted-foreground ring-border/40"
            )}
          >
            {enableAbuseCap ? "⚠ Abuse cap ON" : "⚠ Abuse cap OFF"}
          </button>
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-[10px] text-muted-foreground shrink-0">
              {abuseCaps.size} guild{abuseCaps.size !== 1 ? "s" : ""} affected:
            </span>
            {abuseCaps.size === 0 ? (
              <span className="text-[10px] text-muted-foreground/50">none</span>
            ) : (
              Array.from(abuseCaps.keys()).map((tid) => {
                const g = guilds.find((g) => threadID(g) === tid)
                if (!g) { return null }
                return (
                  <span
                    key={tid}
                    title={`Thread ${tid} — ${abuseCaps.get(tid)!.size} voter(s) capped`}
                    className="inline-flex items-center rounded-full px-1.5 py-0.5 text-[10px] font-medium bg-amber-500/10 text-amber-300 ring-1 ring-inset ring-amber-500/30"
                  >
                    {stripGuildShowcase(g.guildName || g.name)}
                  </span>
                )
              })
            )}
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

      <div className="rounded-xl ring-1 ring-border bg-muted/20 p-4 space-y-3">
        <button
          onClick={() => setVotersOpen((v) => !v)}
          className="flex items-center gap-2 w-full text-left cursor-pointer"
        >
          <span className="text-sm font-semibold">Voters ({voterLeaderboard.length})</span>
          <span className="text-muted-foreground text-xs">{votersOpen ? "▲" : "▼"}</span>
        </button>
        {votersOpen && (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-muted-foreground border-b border-border">
                    <th className="text-left py-1 pr-4 font-medium">#</th>
                    <th className="text-left py-1 pr-4 font-medium">Voter</th>
                    <th className="text-right py-1 pr-4 font-medium">Threads</th>
                    <th className="text-right py-1 pr-4 font-medium">Reactions</th>
                    <th className="text-left py-1 font-medium">Weight</th>
                  </tr>
                </thead>
                <tbody>
                  {voterLeaderboard.slice((voterPage - 1) * VOTER_PAGE_SIZE, voterPage * VOTER_PAGE_SIZE).map((v, i) => (
                    <tr
                      key={v.uid}
                      className="border-b border-border/30 last:border-0 cursor-pointer hover:bg-muted/30 transition-colors"
                      onClick={() => filterByVoter(v.uid)}
                    >
                      <td className="py-1 pr-4 text-muted-foreground/50">{(voterPage - 1) * VOTER_PAGE_SIZE + i + 1}</td>
                      <td className="py-1 pr-4 font-medium">{displayName(v.uid, users)}</td>
                      <td className="py-1 pr-4 text-right font-mono">{v.threads}</td>
                      <td className="py-1 pr-4 text-right font-mono text-muted-foreground">{v.reactions}</td>
                      <td className="py-1">
                        <span className={cn("inline-flex items-center rounded-full px-1.5 py-0.5 text-[10px] font-medium", weightColor(v.weight))}>
                          {weightLabel(v.weight)}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {voterLeaderboard.length > VOTER_PAGE_SIZE && (
              <div className="flex items-center gap-1 pt-1">
                <Button variant="outline" size="xs" onClick={() => setVoterPage((p) => Math.max(1, p - 1))} disabled={voterPage === 1}>←</Button>
                <span className="text-xs px-2 text-muted-foreground">{voterPage} / {Math.ceil(voterLeaderboard.length / VOTER_PAGE_SIZE)}</span>
                <Button variant="outline" size="xs" onClick={() => setVoterPage((p) => Math.min(Math.ceil(voterLeaderboard.length / VOTER_PAGE_SIZE), p + 1))} disabled={voterPage === Math.ceil(voterLeaderboard.length / VOTER_PAGE_SIZE)}>→</Button>
              </div>
            )}
          </>
        )}
      </div>

      <div className="flex items-center gap-3 flex-wrap">
        <Input
          type="text"
          placeholder="Filter by voter name…"
          value={filterVoter}
          onChange={(e) => {
            setFilterVoter(e.target.value)
            setSelectedVoterUid(null)
            setPage(1)
          }}
          className="h-8 w-64 text-sm"
        />
        {selectedVoterUid && (
          <div className="flex items-center gap-1.5 rounded-md bg-muted/40 px-2 py-1 ring-1 ring-border">
            <span className="font-mono text-xs text-muted-foreground select-all">{selectedVoterUid}</span>
            <button
              onClick={() => copyUid(selectedVoterUid)}
              className="text-muted-foreground/50 hover:text-muted-foreground transition-colors text-xs leading-none"
              title="Copy user ID"
            >
              {copiedUid ? "✓" : "⎘"}
            </button>
          </div>
        )}
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
                                      onClick={() => filterByVoter(voter)}
                                      title={`${name} (${voter}) — ${voterCounts.get(voter) ?? 0} distinct threads, weight ×${w}`}
                                      className={cn(
                                        "inline-flex items-center rounded-full px-1.5 py-0.5 text-[10px] font-medium cursor-pointer",
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
