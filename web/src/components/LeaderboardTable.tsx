import { useState, useMemo, useEffect, useRef, useSyncExternalStore } from "react"
import * as React from "react"
import type { RankedGuild } from "@/types/guild"
import { getTier } from "@/lib/scoring"
import { formatBuilderName, stripGuildShowcase, thumbUrl } from "@/lib/format"
import { url } from "@/lib/url"
import { cn } from "@/lib/utils"
import { isCommunityPosted } from "@/lib/config"
import { parseLastModified, formatLastModified } from "@/lib/dates"
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
import { Skeleton } from "@/components/ui/skeleton"
import { HoverCard, HoverCardTrigger, HoverCardContent } from "@/components/ui/hover-card"
import {
  Sheet,
  SheetTrigger,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetFooter,
} from "@/components/ui/sheet"
import { Search, X, SlidersHorizontal, ChevronDown, ChevronRight } from "lucide-react"


type SortField = "rank" | "name" | "lastUpdated"
type SortDir = "asc" | "desc"

interface Props {
  guilds: RankedGuild[]
  allTags: string[]
  basePath?: string
}

function formatBuilders(builders: string[] | undefined): string {
  const names = (builders ?? []).map(formatBuilderName).filter(Boolean)
  const s = names.join(", ") || "—"
  if (s.length <= 50) {
    return s
  }
  return s.slice(0, 50).replace(/,?\s*\w*$/, "") + "..."
}

const TAG_PALETTE = [
  { base: "bg-violet-500/10 text-violet-700 dark:text-violet-300 ring-1 ring-inset ring-violet-500/30", active: "bg-violet-600 text-white ring-1 ring-inset ring-violet-700/50 dark:bg-violet-500" },
  { base: "bg-sky-500/10 text-sky-700 dark:text-sky-300 ring-1 ring-inset ring-sky-500/30", active: "bg-sky-600 text-white ring-1 ring-inset ring-sky-700/50 dark:bg-sky-500" },
  { base: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 ring-1 ring-inset ring-emerald-500/30", active: "bg-emerald-600 text-white ring-1 ring-inset ring-emerald-700/50 dark:bg-emerald-500" },
  { base: "bg-amber-500/10 text-amber-700 dark:text-amber-300 ring-1 ring-inset ring-amber-500/30", active: "bg-amber-600 text-white ring-1 ring-inset ring-amber-700/50 dark:bg-amber-500" },
  { base: "bg-rose-500/10 text-rose-700 dark:text-rose-300 ring-1 ring-inset ring-rose-500/30", active: "bg-rose-600 text-white ring-1 ring-inset ring-rose-700/50 dark:bg-rose-500" },
  { base: "bg-teal-500/10 text-teal-700 dark:text-teal-300 ring-1 ring-inset ring-teal-500/30", active: "bg-teal-600 text-white ring-1 ring-inset ring-teal-700/50 dark:bg-teal-500" },
  { base: "bg-orange-500/10 text-orange-700 dark:text-orange-300 ring-1 ring-inset ring-orange-500/30", active: "bg-orange-600 text-white ring-1 ring-inset ring-orange-700/50 dark:bg-orange-500" },
  { base: "bg-indigo-500/10 text-indigo-700 dark:text-indigo-300 ring-1 ring-inset ring-indigo-500/30", active: "bg-indigo-600 text-white ring-1 ring-inset ring-indigo-700/50 dark:bg-indigo-500" },
]

function tagPalette(label: string) {
  const hash = [...label].reduce((a, c) => a + c.charCodeAt(0), 0)
  return TAG_PALETTE[hash % TAG_PALETTE.length]
}

function Tag({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  const cfg = tagPalette(label)
  return (
    <button
      type="button"
      onClick={(e) => { e.stopPropagation(); onClick() }}
      className={cn("inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium cursor-pointer transition-colors border-0", active ? cfg.active : cfg.base)}
    >
      {label}
    </button>
  )
}

function SortButton({
  field,
  current,
  dir,
  onClick,
  children,
}: {
  field: SortField
  current: SortField
  dir: SortDir
  onClick: (f: SortField) => void
  children: React.ReactNode
}) {
  const active = current === field
  if (field === "rank") {
    return (
      <button
        onClick={() => onClick(field)}
        className={cn(
          "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium transition-all border-0",
          active
            ? "text-foreground ring-2 ring-inset ring-border/80"
            : "text-muted-foreground ring-1 ring-inset ring-border/50 hover:ring-border hover:text-foreground",
        )}
        style={active ? { background: "linear-gradient(135deg, rgba(168,85,247,0.10) 0%, rgba(34,211,238,0.10) 100%)" } : undefined}
      >
        <span
          className={cn("size-1.5 rounded-full shrink-0 transition-colors", active ? "bg-fuchsia-400" : "bg-muted-foreground/40")}
        />
        {children}
        <span className="text-[9px] opacity-60">{active ? (dir === "asc" ? "▲" : "▼") : "⇅"}</span>
      </button>
    )
  }
  return (
    <button
      onClick={() => onClick(field)}
      className="inline-flex items-center gap-1 hover:text-foreground transition-colors"
    >
      {children}
      <span className="text-[10px] opacity-50">{active ? (dir === "asc" ? "▲" : "▼") : "⇅"}</span>
    </button>
  )
}

const PODIUM_ROW: Record<number, string> = {
  1: "bg-yellow-400/8 hover:bg-yellow-400/14",
  2: "bg-slate-400/6 hover:bg-slate-400/12",
  3: "bg-orange-400/6 hover:bg-orange-400/12",
}

const PAGE_SIZE = 40

function SingleGuildRow({ g, gi, guildsLength, basePath, isSolos, activeTags, toggleTag }: {
  g: RankedGuild
  gi: number
  guildsLength: number
  basePath: string
  isSolos: boolean
  activeTags: Set<string>
  toggleTag: (tag: string) => void
}) {
  const [open, setOpen] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout>>()
  const img = g.coverImage ?? g.screenshots?.[0]
  const podium = PODIUM_ROW[g.rank]
  const tier = getTier(g.rank, guildsLength, g.score)
  const fmt = formatLastModified(g.lastModified)

  const enter = () => {
    clearTimeout(timer.current)
    timer.current = setTimeout(() => setOpen(true), 250)
  }
  const leave = () => {
    clearTimeout(timer.current)
    timer.current = setTimeout(() => setOpen(false), 100)
  }

  return (
    <HoverCard open={open} onOpenChange={() => {}} openDelay={0} closeDelay={0}>
      <TableRow
        key={g.slug}
        onMouseEnter={enter}
        onMouseLeave={leave}
        onClick={() => {
          ;(window as any).umami?.track("guild_click", { name: g.guildName || g.name, rank: g.rank, source: "table", type: basePath })
          window.location.href = url(`/${basePath}/${g.slug}`)
        }}
        className={cn("cursor-pointer transition-colors", podium ?? (gi % 2 !== 0 ? "bg-muted/10 hover:bg-muted/20" : "hover:bg-muted/10"))}
      >
        {!isSolos && (
          <TableCell className="text-center">
            <span className={cn("inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs", tier.badge)} style={tier.badgeStyle}>
              <span className={cn("size-1.5 rounded-full shrink-0", tier.dot)} />
              {tier.label}
            </span>
          </TableCell>
        )}
        <TableCell>
          <div className="flex items-center gap-2.5">
            {img && (
              <img
                src={thumbUrl(img, 120, 120)}
                alt={stripGuildShowcase(g.guildName || g.name)}
                className="w-8 h-8 rounded-md object-cover shrink-0 hidden sm:block"
                loading="lazy"
                onError={(e) => ((e.target as HTMLImageElement).style.display = "none")}
              />
            )}
            <HoverCardTrigger asChild>
              <div className="min-w-0">
                <a href={url(`/${basePath}/${g.slug}`)} className="font-medium hover:underline" onClick={(e) => e.stopPropagation()}>
                  {stripGuildShowcase(g.guildName || g.name)}
                </a>
                {g.buildTitle && (
                  <p className="text-[11px] text-muted-foreground/60 leading-tight truncate">{g.buildTitle}{g.isCurrent && <span className="ml-1 text-emerald-500/80">●</span>}</p>
                )}
              </div>
            </HoverCardTrigger>
          </div>
        </TableCell>
        <TableCell className="text-muted-foreground hidden md:table-cell">
          <div className="flex items-center gap-1.5">
            {formatBuilders(g.builders)}
            {isCommunityPosted(g) && <span title={g.postedOnBehalfOf ? `Posted on behalf of @${g.postedOnBehalfOf}` : "Submitted by the community"} className="size-1.5 rounded-full bg-sky-400/60 shrink-0" />}
          </div>
        </TableCell>
        <TableCell className="hidden lg:table-cell">
          <div className="flex flex-wrap gap-1">{g.tags?.map((tag) => <Tag key={tag} label={tag} active={activeTags.has(tag)} onClick={() => toggleTag(tag)} />)}</div>
        </TableCell>
        <TableCell className="hidden lg:table-cell text-[11px] text-muted-foreground/50">
          {fmt ? <span title={fmt.full} className="cursor-default">{fmt.relative}</span> : <span>—</span>}
        </TableCell>
      </TableRow>
      <HoverCardContent className="w-72 p-0 overflow-hidden" side="right" align="start" onMouseEnter={enter} onMouseLeave={leave}>
        {img && <div className="aspect-video w-full overflow-hidden"><img src={thumbUrl(img, 400, 225)} alt={stripGuildShowcase(g.guildName || g.name)} className="w-full h-full object-cover" /></div>}
        <div className="p-3">
          <p className="font-medium text-sm leading-tight">{stripGuildShowcase(g.guildName || g.name)}</p>
          {g.builders && g.builders.length > 0 && <p className="text-xs text-muted-foreground mt-0.5">by {formatBuilders(g.builders)}</p>}
          {g.tags && g.tags.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-2">{g.tags.slice(0, 5).map((tag) => <Tag key={tag} label={tag} active={activeTags.has(tag)} onClick={() => toggleTag(tag)} />)}</div>
          )}
        </div>
      </HoverCardContent>
    </HoverCard>
  )
}

export function LeaderboardTable({ guilds, allTags, basePath = "guilds" }: Props) {
  const isSolos = basePath === "solos"
  const [sortField, setSortField] = useState<SortField>("rank")
  const [sortDir, setSortDir] = useState<SortDir>("asc")
  const [activeTags, setActiveTags] = useState<Set<string>>(() => {
    if (typeof window === "undefined") { return new Set() }
    const p = new URLSearchParams(window.location.search)
    const tags = p.get("tags")
    return tags ? new Set(tags.split(",").filter(Boolean)) : new Set()
  })
  const [search, setSearch] = useState(() => {
    if (typeof window === "undefined") { return "" }
    return new URLSearchParams(window.location.search).get("q") ?? ""
  })
  const [page, setPage] = useState(1)
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set())
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isFirstRender = useRef(true)
  const inputRef = useRef<HTMLInputElement>(null)
  const mounted = useSyncExternalStore(() => () => {}, () => true, () => false)

  useEffect(() => {
    if (search) {
      inputRef.current?.scrollIntoView({ behavior: "smooth", block: "start" })
    }
    // intentionally mount-only: scroll once if ?q= was in URL
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (isFirstRender.current) { isFirstRender.current = false; return }
    const p = new URLSearchParams(window.location.search)
    if (search) { p.set("q", search) } else { p.delete("q") }
    if (activeTags.size > 0) { p.set("tags", [...activeTags].join(",")) } else { p.delete("tags") }
    history.replaceState(null, "", p.toString() ? `?${p}` : location.pathname)
  }, [search, activeTags])

  function toggleSort(field: SortField) {
    if (sortField === field) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"))
    } else {
      setSortField(field)
      setSortDir(field === "rank" ? "asc" : "desc")
    }
    setPage(1)
  }

  function toggleGroup(name: string) {
    setExpandedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(name)) { next.delete(name) } else { next.add(name) }
      return next
    })
  }

  function toggleTag(tag: string) {
    setActiveTags((prev) => {
      const next = new Set(prev)
      if (next.has(tag)) {
        next.delete(tag)
      } else {
        next.add(tag)
        ;window.umami?.track("tag_filter", { tag, type: basePath })
      }
      return next
    })
    setPage(1)
  }

  const deaccent = (s: string) => {
    const charMap: Record<string, string> = {
      'ð': 'd', 'þ': 'th', 'ø': 'o', 'ß': 'ss', 'æ': 'ae',
    }
    let result = s.normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase()
    for (const [char, replacement] of Object.entries(charMap)) {
      result = result.replaceAll(char, replacement)
    }
    return result
  }

  const filtered = useMemo(() => {
    const q = deaccent(search.trim())
    let result = guilds
    if (activeTags.size > 0) {
      result = result.filter((g) => g.tags?.some((t) => activeTags.has(t)))
    }
    if (q) {
      result = result.filter((g) =>
        deaccent(g.guildName || g.name).includes(q) ||
        (g.buildTitle ? deaccent(g.buildTitle).includes(q) : false) ||
        (g.builders ?? []).some((b) => deaccent(formatBuilderName(b)).includes(q))
      )
    }

    return [...result].sort((a, b) => {
      let cmp = 0
      if (sortField === "rank") {
        cmp = a.rank - b.rank
      } else if (sortField === "name") {
        cmp = a.name.localeCompare(b.name)
      } else if (sortField === "lastUpdated") {
        cmp = parseLastModified(a.lastModified) - parseLastModified(b.lastModified)
      }
      return sortDir === "asc" ? cmp : -cmp
    })
  }, [guilds, activeTags, sortField, sortDir, search])

  const groupedGuilds = useMemo(() => {
    if (isSolos) {
      return filtered.map((g) => ({ key: g.slug, builds: [g], isMulti: false }))
    }
    const nameToBuilds = new Map<string, RankedGuild[]>()
    for (const g of filtered) {
      if (!nameToBuilds.has(g.name)) nameToBuilds.set(g.name, [])
      nameToBuilds.get(g.name)!.push(g)
    }
    const groups: Array<{ key: string; builds: RankedGuild[]; isMulti: boolean }> = []
    for (const [key, builds] of nameToBuilds) {
      groups.push({ key, builds, isMulti: builds.length > 1 })
    }
    groups.sort((a, b) => {
      let cmp = 0
      if (sortField === "rank") {
        cmp = Math.min(...a.builds.map((g) => g.rank)) - Math.min(...b.builds.map((g) => g.rank))
      } else if (sortField === "name") {
        cmp = a.key.localeCompare(b.key)
      } else if (sortField === "lastUpdated") {
        cmp = Math.max(...a.builds.map((g) => parseLastModified(g.lastModified))) -
              Math.max(...b.builds.map((g) => parseLastModified(g.lastModified)))
      }
      return sortDir === "asc" ? cmp : -cmp
    })
    return groups
  }, [filtered, isSolos, sortField, sortDir])

  const totalPages = Math.ceil(groupedGuilds.length / PAGE_SIZE)
  const paginatedGroups = groupedGuilds.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  if (!mounted) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-9 w-full sm:max-w-lg rounded-lg" />
        <div className="rounded-xl ring-1 ring-border overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/40">
                {!isSolos && <TableHead className="w-24" />}
                <TableHead />
                <TableHead className="hidden md:table-cell" />
                <TableHead className="hidden lg:table-cell" />
                <TableHead className="hidden lg:table-cell w-28" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {Array.from({ length: 8 }).map((_, i) => (
                <TableRow key={i}>
                  {!isSolos && <TableCell><Skeleton className="h-6 w-16 rounded-full mx-auto" /></TableCell>}
                  <TableCell>
                    <div className="flex items-center gap-2.5">
                      <Skeleton className="w-8 h-8 rounded-md shrink-0 hidden sm:block" />
                      <Skeleton className="h-4 w-32" />
                    </div>
                  </TableCell>
                  <TableCell className="hidden md:table-cell"><Skeleton className="h-4 w-28" /></TableCell>
                  <TableCell className="hidden lg:table-cell"><Skeleton className="h-5 w-20 rounded-full" /></TableCell>
                  <TableCell className="hidden lg:table-cell"><Skeleton className="h-4 w-14" /></TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-1.5">
        <div className="relative w-full sm:max-w-lg">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none" />
          <Input
            ref={inputRef}
            type="text"
            placeholder={isSolos ? "Search bases or builders…" : "Search guilds or builders…"}
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPage(1)
              if (searchTimerRef.current) { clearTimeout(searchTimerRef.current) }
              if (e.target.value.trim()) {
                searchTimerRef.current = setTimeout(() => {
                  ;window.umami?.track("search_used", { query: e.target.value.trim(), page: basePath })
                }, 1500)
              }
            }}
            className="pl-9 pr-8 focus-visible:border-primary focus-visible:ring-primary/30"
          />
          {search && (
            <button
              onClick={() => { setSearch(""); setPage(1) }}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
              aria-label="Clear search"
            >
              <X className="size-3.5" />
            </button>
          )}
        </div>
        <p className="text-xs text-muted-foreground">
          Need something more specific? Ask on <span className="font-medium text-foreground">#ruby</span> on our Discord, she can pinpoint exactly what you need (Pikachu, cats, horse racing...)!
        </p>
      </div>

      {allTags.length > 0 && (
        <div>
          {/* Mobile: filter sheet trigger */}
          <div className="flex sm:hidden items-center gap-2">
            <Sheet>
              <SheetTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5 h-8">
                  <SlidersHorizontal className="size-3.5" />
                  Filters{activeTags.size > 0 ? ` (${activeTags.size})` : ""}
                </Button>
              </SheetTrigger>
              <SheetContent side="bottom" className="max-h-[70vh]">
                <SheetHeader>
                  <SheetTitle>Filter by tag</SheetTitle>
                </SheetHeader>
                <div className="flex flex-wrap gap-2 px-4 py-3 overflow-y-auto">
                  {allTags.map((tag) => (
                    <Tag key={tag} label={tag} active={activeTags.has(tag)} onClick={() => toggleTag(tag)} />
                  ))}
                </div>
                {activeTags.size > 0 && (
                  <SheetFooter className="px-4 pb-4">
                    <Button variant="ghost" size="sm" onClick={() => { setActiveTags(new Set()); setPage(1) }}>
                      Clear all filters
                    </Button>
                  </SheetFooter>
                )}
              </SheetContent>
            </Sheet>
            {activeTags.size > 0 && (
              <span className="text-xs text-muted-foreground">{activeTags.size} active</span>
            )}
          </div>
          {/* Desktop: inline tags */}
          <div className="hidden sm:flex flex-wrap gap-1.5 items-center">
            <span className="text-xs text-muted-foreground mr-1">Filter:</span>
            {allTags.map((tag) => (
              <Tag key={tag} label={tag} active={activeTags.has(tag)} onClick={() => toggleTag(tag)} />
            ))}
            {activeTags.size > 0 && (
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={() => { setActiveTags(new Set()); setPage(1) }}
                title="Clear filters"
                aria-label="Clear filters"
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </Button>
            )}
          </div>
        </div>
      )}

      {!isSolos && (
        <p className="text-xs text-muted-foreground/70 italic">
          Scores don't tell the whole story. Plenty of hidden gems await even with low score, but you'll have to earn them... {" "}
          <a href={url("/how-it-works")} className="underline underline-offset-2 hover:text-foreground transition-colors not-italic">How scoring works ↗</a>
        </p>
      )}

      {totalPages > 1 && (
        <div className="flex items-center gap-1">
          <Button variant="outline" size="xs" className="cursor-pointer" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1}>←</Button>
          {Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
            <Button key={p} variant={p === page ? "default" : "outline"} size="xs" className="cursor-pointer" onClick={() => setPage(p)}>{p}</Button>
          ))}
          <Button variant="outline" size="xs" className="cursor-pointer" onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page === totalPages}>→</Button>
        </div>
      )}

      <div className="rounded-xl ring-1 ring-border overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-muted/40 text-muted-foreground">
              {!isSolos && (
                <TableHead className="w-24">
                  <SortButton field="rank" current={sortField} dir={sortDir} onClick={toggleSort}>
                    Tier
                  </SortButton>
                </TableHead>
              )}
              <TableHead>
                <SortButton field="name" current={sortField} dir={sortDir} onClick={toggleSort}>
                  {isSolos ? "Build" : "Guild"}
                </SortButton>
              </TableHead>
              <TableHead className="hidden md:table-cell">Builders</TableHead>
              <TableHead className="hidden lg:table-cell">Tags</TableHead>
              <TableHead className="hidden lg:table-cell w-28">
                <SortButton field="lastUpdated" current={sortField} dir={sortDir} onClick={toggleSort}>
                  Last updated
                </SortButton>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {paginatedGroups.length === 0 ? (
              <TableRow>
                <TableCell colSpan={isSolos ? 4 : 5} className="py-8 text-center text-muted-foreground">
                  {isSolos ? "No bases match your search." : "No guilds match your search."}
                </TableCell>
              </TableRow>
            ) : paginatedGroups.flatMap((group, gi) => {
              const { key, builds, isMulti } = group
              const isExpanded = expandedGroups.has(key)

              function renderTierBadge(g: RankedGuild) {
                const tier = getTier(g.rank, guilds.length, g.score)
                return (
                  <span className={cn("inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs", tier.badge)} style={tier.badgeStyle}>
                    <span className={cn("size-1.5 rounded-full shrink-0", tier.dot)} />
                    {tier.label}
                  </span>
                )
              }

              function renderLastUpdated(g: RankedGuild) {
                const fmt = formatLastModified(g.lastModified)
                return fmt ? <span title={fmt.full} className="cursor-default">{fmt.relative}</span> : <span>—</span>
              }

              if (!isMulti) {
                return [<SingleGuildRow key={builds[0].slug} g={builds[0]} gi={gi} guildsLength={guilds.length} basePath={basePath} isSolos={isSolos} activeTags={activeTags} toggleTag={toggleTag} />]
              }

              // Multi-build guild
              const best = builds[0]
              const bestImg = best.coverImage ?? best.screenshots?.[0]
              const allTags = [...new Set(builds.flatMap((g) => g.tags ?? []))]
              const latestBuild = builds.reduce((a, b) => parseLastModified(a.lastModified) >= parseLastModified(b.lastModified) ? a : b)
              const podium = PODIUM_ROW[best.rank]

              const headerRow = (
                <TableRow
                  key={`group-${key}`}
                  onClick={() => toggleGroup(key)}
                  className={cn("cursor-pointer select-none transition-colors", podium ?? (gi % 2 !== 0 ? "bg-muted/10 hover:bg-muted/20" : "hover:bg-muted/10"))}
                >
                  {!isSolos && <TableCell className="text-center">{renderTierBadge(best)}</TableCell>}
                  <TableCell>
                    <div className="flex items-center gap-2.5">
                      {bestImg && <img src={thumbUrl(bestImg, 120, 120)} alt={stripGuildShowcase(best.guildName || best.name)} className="w-8 h-8 rounded-md object-cover shrink-0 hidden sm:block" loading="lazy" onError={(e) => ((e.target as HTMLImageElement).style.display = "none")} />}
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-medium">{stripGuildShowcase(best.guildName || best.name)}</span>
                          <span className="inline-flex items-center gap-0.5 text-[10px] font-medium px-1.5 py-0.5 rounded-full bg-primary/8 text-primary/70 ring-1 ring-inset ring-primary/20 shrink-0">
                            {isExpanded ? <ChevronDown className="size-2.5" /> : <ChevronRight className="size-2.5" />}
                            {builds.length} builds
                          </span>
                        </div>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell className="text-muted-foreground hidden md:table-cell">{formatBuilders(best.builders)}</TableCell>
                  <TableCell className="hidden lg:table-cell">
                    <div className="flex flex-wrap gap-1">{allTags.slice(0, 4).map((tag) => <Tag key={tag} label={tag} active={activeTags.has(tag)} onClick={() => toggleTag(tag)} />)}</div>
                  </TableCell>
                  <TableCell className="hidden lg:table-cell text-[11px] text-muted-foreground/50">{renderLastUpdated(latestBuild)}</TableCell>
                </TableRow>
              )

              if (!isExpanded) return [headerRow]

              const buildRows = builds.map((g, bi) => {
                const img = g.coverImage ?? g.screenshots?.[0]
                return (
                  <TableRow
                    key={g.slug}
                    onClick={() => {
                      ;window.umami?.track("guild_click", { name: g.guildName || g.name, rank: g.rank, source: "table_build", type: basePath })
                      window.location.href = url(`/${basePath}/${g.slug}`)
                    }}
                    className={cn("cursor-pointer transition-colors border-l-2 border-l-primary/20", bi % 2 === 0 ? "bg-muted/5 hover:bg-muted/15" : "bg-muted/10 hover:bg-muted/20")}
                  >
                    {!isSolos && <TableCell />}
                    <TableCell>
                      <div className="flex items-center gap-2.5 pl-4">
                        {img && <img src={thumbUrl(img, 120, 120)} alt={g.buildTitle || "Default"} className="w-8 h-8 rounded-md object-cover shrink-0 hidden sm:block" loading="lazy" onError={(e) => ((e.target as HTMLImageElement).style.display = "none")} />}
                        <div className="min-w-0">
                          <div className="flex items-center gap-1.5 flex-wrap">
                            <a href={url(`/${basePath}/${g.slug}`)} className="text-sm font-medium hover:underline" onClick={(e) => e.stopPropagation()}>
                              {g.buildTitle || "Default"}
                            </a>
                            {g.isCurrent && <span className="text-[10px] text-emerald-500/90">● current</span>}
                            {(() => { const t = getTier(g.rank, guilds.length, g.score); return (
                              <span className={cn("inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-[10px] opacity-55", t.badge)} style={t.badgeStyle}>
                                <span className={cn("size-1 rounded-full shrink-0", t.dot)} />{t.label}
                              </span>
                            )})()}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground hidden md:table-cell">
                      <div className="flex items-center gap-1.5">
                        {formatBuilders(g.builders)}
                        {isCommunityPosted(g) && <span title={g.postedOnBehalfOf ? `Posted on behalf of @${g.postedOnBehalfOf}` : "Submitted by the community"} className="size-1.5 rounded-full bg-sky-400/60 shrink-0" />}
                      </div>
                    </TableCell>
                    <TableCell className="hidden lg:table-cell">
                      <div className="flex flex-wrap gap-1">{g.tags?.map((tag) => <Tag key={tag} label={tag} active={activeTags.has(tag)} onClick={() => toggleTag(tag)} />)}</div>
                    </TableCell>
                    <TableCell className="hidden lg:table-cell text-[11px] text-muted-foreground/50">{renderLastUpdated(g)}</TableCell>
                  </TableRow>
                )
              })

              return [headerRow, ...buildRows]
            })}
          </TableBody>
        </Table>
      </div>
      <div className="flex items-center justify-between">
        {totalPages > 1 ? (
          <div className="flex items-center gap-1">
            <Button variant="outline" size="xs" className="cursor-pointer" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1}>←</Button>
            {Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
              <Button key={p} variant={p === page ? "default" : "outline"} size="xs" className="cursor-pointer" onClick={() => setPage(p)}>{p}</Button>
            ))}
            <Button variant="outline" size="xs" className="cursor-pointer" onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page === totalPages}>→</Button>
          </div>
        ) : <span />}
        <p className="text-xs text-muted-foreground">
          {filtered.length} / {guilds.length} {isSolos ? "bases" : "guilds"}
        </p>
      </div>
    </div>
  )
}
