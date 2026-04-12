import { useState, useMemo } from "react"
import * as React from "react"
import type { RankedGuild } from "@/types/guild"
import { rankLabel } from "@/lib/slugify"
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
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Search, X } from "lucide-react"

type SortField = "rank" | "name" | "score"
type SortDir = "asc" | "desc"

interface Props {
  guilds: RankedGuild[]
  allTags: string[]
  basePath?: string
}

function formatBuilders(builders: string[] | undefined): string {
  const s = (builders ?? []).join(", ") || "—"
  if (s.length <= 50) {
    return s
  }
  return s.slice(0, 50).replace(/,?\s*\w*$/, "") + "..."
}

const TAG_COLORS: Record<string, [number, number]> = {
  "Arena":          [30,  70],
  "Castle":         [220, 40],
  "Cave":           [25,  30],
  "City":           [210, 15],
  "Desert":         [40,  65],
  "Floating island":[195, 60],
  "Fun":            [320, 60],
  "Maze":           [270, 45],
  "Military":       [80,  35],
  "Mountain":       [175, 30],
  "Nature":         [130, 45],
  "River":          [205, 60],
  "Snow":           [200, 50],
  "Zen":            [100, 25],
}

function tagColor(label: string): [number, number] {
  if (TAG_COLORS[label]) return TAG_COLORS[label]
  let hash = 0
  for (let i = 0; i < label.length; i++) hash = (hash * 31 + label.charCodeAt(i)) >>> 0
  return [hash % 360, 50]
}

function Tag({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  const [h, s] = tagColor(label)
  const style = active
    ? { background: `hsl(${h} ${s}% 38%)`, color: "hsl(0 0% 100%)", borderColor: "transparent" }
    : { background: `hsl(${h} ${s}% 88%)`, color: `hsl(${h} ${s - 10}% 28%)`, borderColor: "transparent" }
  return (
    <Badge
      variant="outline"
      onClick={(e) => { e.stopPropagation(); onClick() }}
      className="cursor-pointer transition-colors"
      style={style}
    >
      {label}
    </Badge>
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

const PAGE_SIZE = 40

export function GuildTable({ guilds, allTags, basePath = "guilds" }: Props) {
  const isSolos = basePath === "solos"
  const [sortField, setSortField] = useState<SortField>("rank")
  const [sortDir, setSortDir] = useState<SortDir>("asc")
  const [activeTags, setActiveTags] = useState<Set<string>>(new Set())
  const [search, setSearch] = useState("")
  const [page, setPage] = useState(1)
  const searchTimerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null)

  function toggleSort(field: SortField) {
    if (sortField === field) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"))
    } else {
      setSortField(field)
      setSortDir(field === "rank" ? "asc" : "desc")
    }
    setPage(1)
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

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    let result = guilds
    if (activeTags.size > 0) {
      result = result.filter((g) => g.tags?.some((t) => activeTags.has(t)))
    }
    if (q) {
      result = result.filter((g) =>
        (g.guildName || g.name).toLowerCase().includes(q) ||
        (g.builders ?? []).some((b) => b.toLowerCase().includes(q))
      )
    }

    return [...result].sort((a, b) => {
      let cmp = 0
      if (sortField === "rank") {
        cmp = a.rank - b.rank
      } else if (sortField === "name") {
        cmp = a.name.localeCompare(b.name)
      } else if (sortField === "score") {
        cmp = b.score - a.score
      }
      return sortDir === "asc" ? cmp : -cmp
    })
  }, [guilds, activeTags, sortField, sortDir, search])

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE)
  const paginated = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  return (
    <div className="space-y-4">
      <div className="relative w-full sm:w-96">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none" />
        <Input
          type="search"
          placeholder={isSolos ? "Search bases or builders…" : "Search guilds or builders…"}
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
            setPage(1)
            if (searchTimerRef.current) clearTimeout(searchTimerRef.current)
            if (e.target.value.trim()) {
              searchTimerRef.current = setTimeout(() => {
                ;window.umami?.track("search_used", { page: basePath })
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

      {allTags.length > 0 && (
        <div className="flex flex-wrap gap-1.5 items-center">
          <span className="text-xs text-muted-foreground mr-1">Filter:</span>
          {allTags.map((tag) => (
            <Tag
              key={tag}
              label={tag}
              active={activeTags.has(tag)}
              onClick={() => toggleTag(tag)}
            />
          ))}
          {activeTags.size > 0 && (
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={() => setActiveTags(new Set())}
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
      )}

      <div className="rounded-xl ring-1 ring-border overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-muted/40 text-muted-foreground">
              <TableHead className="w-16">
                <SortButton field="rank" current={sortField} dir={sortDir} onClick={toggleSort}>
                  Rank
                </SortButton>
              </TableHead>
              <TableHead>
                <SortButton field="name" current={sortField} dir={sortDir} onClick={toggleSort}>
                  Guild
                </SortButton>
              </TableHead>
              <TableHead className="hidden md:table-cell">Builders</TableHead>
              <TableHead className="hidden lg:table-cell">Tags</TableHead>
              <TableHead className="w-20 text-right">
                <SortButton field="score" current={sortField} dir={sortDir} onClick={toggleSort}>
                  Score
                </SortButton>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {paginated.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="py-8 text-center text-muted-foreground">
                  {isSolos ? "No bases match your search." : "No guilds match your search."}
                </TableCell>
              </TableRow>
            ) : paginated.map((g, i) => (
              <TableRow
                key={g.name}
                onClick={() => {
                  ;window.umami?.track("guild_click", { name: g.guildName || g.name, rank: g.rank, source: "table", type: basePath })
                  window.location.href = url(`/${basePath}/${g.slug}`)
                }}
                className={cn("cursor-pointer", i % 2 !== 0 && "bg-muted/10")}
              >
                <TableCell className="text-center font-medium">{rankLabel(g.rank)}</TableCell>
                <TableCell className="font-medium">{g.guildName || g.name}</TableCell>
                <TableCell className="text-muted-foreground hidden md:table-cell">
                  {formatBuilders(g.builders)}
                </TableCell>
                <TableCell className="hidden lg:table-cell">
                  <div className="flex flex-wrap gap-1">
                    {g.tags?.map((tag) => (
                      <Tag key={tag} label={tag} active={activeTags.has(tag)} onClick={() => toggleTag(tag)} />
                    ))}
                  </div>
                </TableCell>
                <TableCell className="text-right font-mono font-semibold">{g.score}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      <div className="flex items-center justify-between">
        {totalPages > 1 ? (
          <div className="flex items-center gap-1">
            <Button
              variant="outline"
              size="xs"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
            >
              ←
            </Button>
            {Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
              <Button
                key={p}
                variant={p === page ? "default" : "outline"}
                size="xs"
                onClick={() => setPage(p)}
              >
                {p}
              </Button>
            ))}
            <Button
              variant="outline"
              size="xs"
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
            >
              →
            </Button>
          </div>
        ) : <span />}
        <p className="text-xs text-muted-foreground">
          {filtered.length} / {guilds.length} {isSolos ? "bases" : "guilds"}
        </p>
      </div>
    </div>
  )
}
