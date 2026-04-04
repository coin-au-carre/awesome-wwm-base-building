import { useState, useMemo } from "react"
import type { RankedGuild } from "@/types/guild"
import { rankLabel } from "@/lib/slugify"
import { url } from "@/lib/url"

type SortField = "rank" | "name" | "score"
type SortDir = "asc" | "desc"

interface Props {
  guilds: RankedGuild[]
  allTags: string[]
  basePath?: string
}

function Tag({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={[
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors cursor-pointer",
        active
          ? "bg-primary text-primary-foreground"
          : "bg-muted text-muted-foreground hover:bg-muted/70",
      ].join(" ")}
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

export function GuildTable({ guilds, allTags, basePath = "guilds" }: Props) {
  const [sortField, setSortField] = useState<SortField>("rank")
  const [sortDir, setSortDir] = useState<SortDir>("asc")
  const [activeTags, setActiveTags] = useState<Set<string>>(new Set())
  const [search, setSearch] = useState("")

  function toggleSort(field: SortField) {
    if (sortField === field) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"))
    } else {
      setSortField(field)
      setSortDir(field === "rank" ? "asc" : "desc")
    }
  }

  function toggleTag(tag: string) {
    setActiveTags((prev) => {
      const next = new Set(prev)
      next.has(tag) ? next.delete(tag) : next.add(tag)
      return next
    })
  }

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    let result = guilds
    if (activeTags.size > 0) result = result.filter((g) => g.tags?.some((t) => activeTags.has(t)))
    if (q) result = result.filter((g) => (g.guildName || g.name).toLowerCase().includes(q))

    return [...result].sort((a, b) => {
      let cmp = 0
      if (sortField === "rank") cmp = a.rank - b.rank
      else if (sortField === "name") cmp = a.name.localeCompare(b.name)
      else if (sortField === "score") cmp = b.score - a.score
      return sortDir === "asc" ? cmp : -cmp
    })
  }, [guilds, activeTags, sortField, sortDir, search])

  return (
    <div className="space-y-4">
      {/* Search */}
      <input
        type="search"
        placeholder={basePath === "solos" ? "Search bases…" : "Search guilds…"}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="w-full sm:w-64 rounded-lg border border-border bg-background px-3 py-1.5 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
      />

      {/* Tag filter */}
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
            <button
              onClick={() => setActiveTags(new Set())}
              className="text-muted-foreground hover:text-foreground transition-colors ml-1 cursor-pointer"
              title="Clear filters"
              aria-label="Clear filters"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          )}
        </div>
      )}

      {/* Table */}
      <div className="overflow-x-auto rounded-xl ring-1 ring-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/40 text-muted-foreground text-left">
              <th className="px-4 py-3 font-medium w-16">
                <SortButton field="rank" current={sortField} dir={sortDir} onClick={toggleSort}>
                  Rank
                </SortButton>
              </th>
              <th className="px-4 py-3 font-medium">
                <SortButton field="name" current={sortField} dir={sortDir} onClick={toggleSort}>
                  Guild
                </SortButton>
              </th>
              <th className="px-4 py-3 font-medium hidden md:table-cell">Builders</th>
              <th className="px-4 py-3 font-medium hidden lg:table-cell">Tags</th>
              <th className="px-4 py-3 font-medium w-20 text-right">
                <SortButton field="score" current={sortField} dir={sortDir} onClick={toggleSort}>
                  Score
                </SortButton>
              </th>
            </tr>
          </thead>
          <tbody>
            {filtered.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-sm text-muted-foreground">
                  {basePath === "solos" ? "No bases match your search." : "No guilds match your search."}
                </td>
              </tr>
            ) : filtered.map((g, i) => (
              <tr
                key={g.name}
                onClick={() => window.location.href = url(`/${basePath}/${g.slug}`)}
                className={[
                  "border-b border-border last:border-0 transition-colors hover:bg-muted/30 cursor-pointer",
                  i % 2 === 0 ? "" : "bg-muted/10",
                ].join(" ")}
              >
                <td className="px-4 py-3 text-center font-medium">{rankLabel(g.rank)}</td>
                <td className="px-4 py-3">
                  <span className="font-medium">{g.guildName || g.name}</span>
                </td>
                <td className="px-4 py-3 text-muted-foreground hidden md:table-cell">
                  {(() => { const s = (g.builders ?? []).join(", ") || "—"; return s.length > 50 ? s.slice(0, 50).replace(/,?\s*\w*$/, "") + "..." : s; })()}
                </td>
                <td className="px-4 py-3 hidden lg:table-cell">
                  <div className="flex flex-wrap gap-1">
                    {g.tags?.map((tag) => (
                      <span
                        key={tag}
                        className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                </td>
                <td className="px-4 py-3 text-right font-mono font-semibold">{g.score}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="text-xs text-muted-foreground text-right">
        {filtered.length} / {guilds.length} guilds
      </p>
    </div>
  )
}
