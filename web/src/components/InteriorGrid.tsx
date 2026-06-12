import { useState, useMemo } from "react"
import type { IndexedInterior } from "@/types/interior"
import { parseLastModified, relativeTime } from "@/lib/dates"
import { url } from "@/lib/url"
import { cn } from "@/lib/utils"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Search, X, SlidersHorizontal } from "lucide-react"
import {
  Sheet,
  SheetTrigger,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetFooter,
} from "@/components/ui/sheet"

type SortOrder = "newest" | "oldest"

function interiorDate(it: IndexedInterior, field: "newest" | "oldest"): number {
  const ms = parseLastModified(it.createdAt)
  return field === "newest" ? -ms : ms
}

interface Props {
  interiors: IndexedInterior[]
}

export function InteriorGrid({ interiors }: Props) {
  const [search, setSearch] = useState("")
  const [sortOrder, setSortOrder] = useState<SortOrder>("newest")
  const [sheetOpen, setSheetOpen] = useState(false)

  const filtered = useMemo(() => {
    let result = interiors
    if (search.trim()) {
      const q = search.toLowerCase()
      result = result.filter((it) => it.name.toLowerCase().includes(q))
    }
    return [...result].sort((a, b) => interiorDate(a, sortOrder) - interiorDate(b, sortOrder))
  }, [interiors, search, sortOrder])

  const sortLabels: Record<SortOrder, string> = { newest: "Newest", oldest: "Oldest" }

  const SortPills = (
    <div className="flex flex-wrap gap-1">
      {(["newest", "oldest"] as SortOrder[]).map((s) => (
        <button
          key={s}
          type="button"
          onClick={() => setSortOrder(s)}
          className={cn(
            "rounded-full px-3 py-1 text-xs font-medium transition-colors border",
            sortOrder === s
              ? "bg-primary text-primary-foreground border-primary"
              : "bg-transparent text-muted-foreground border-border hover:text-foreground"
          )}
        >
          {sortLabels[s]}
        </button>
      ))}
    </div>
  )

  return (
    <div className="space-y-4">
      {/* Search + sort bar */}
      <div className="flex gap-2 items-center">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by title…"
            className="pl-8 pr-8"
            data-umami-event="interior_search"
          />
          {search && (
            <button
              type="button"
              onClick={() => setSearch("")}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
            >
              <X className="size-4" />
            </button>
          )}
        </div>

        {/* Sort pills — desktop */}
        <div className="hidden sm:flex items-center gap-1.5">
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60 pr-0.5">Sort</span>
          {(["newest", "oldest"] as SortOrder[]).map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => setSortOrder(s)}
              className={cn(
                "rounded-full px-3 py-1 text-xs font-medium transition-colors border",
                sortOrder === s
                  ? "bg-primary text-primary-foreground border-primary"
                  : "bg-transparent text-muted-foreground border-border hover:text-foreground"
              )}
            >
              {sortLabels[s]}
            </button>
          ))}
        </div>

        {/* Mobile: sort sheet */}
        <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
          <SheetTrigger asChild>
            <Button variant="outline" size="sm" className="sm:hidden shrink-0 gap-1.5">
              <SlidersHorizontal className="size-3.5" />
              Sort
            </Button>
          </SheetTrigger>
          <SheetContent side="bottom" className="max-h-[50dvh] overflow-y-auto rounded-t-2xl">
            <SheetHeader className="mb-4">
              <SheetTitle>Sort interiors</SheetTitle>
            </SheetHeader>
            <div className="pb-2">{SortPills}</div>
            <SheetFooter className="mt-4">
              <Button onClick={() => setSheetOpen(false)}>Done</Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      </div>

      {/* Results count */}
      {search && (
        <p className="text-sm text-muted-foreground">
          {filtered.length} interior{filtered.length !== 1 ? "s" : ""}
        </p>
      )}

      {/* Grid */}
      {filtered.length === 0 ? (
        <p className="text-sm text-muted-foreground py-8 text-center">No interiors match your search.</p>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((it) => {
            const img = it.screenshots?.[0]
            return (
              <a
                key={it.discordThread}
                href={url(`/interiors/${it.slug}`)}
                className="group relative flex flex-col overflow-hidden rounded-xl ring-1 ring-border bg-card hover:ring-primary transition-all"
                data-umami-event="interior_click"
                data-umami-event-name={it.name}
              >
                {/* Thumbnail */}
                <div className="relative aspect-video bg-muted overflow-hidden">
                  {img ? (
                    <img
                      src={img}
                      alt={it.name}
                      loading="lazy"
                      onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = "none" }}
                      className="absolute inset-0 h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                    />
                  ) : (
                    <div className="absolute inset-0 flex items-center justify-center text-muted-foreground/30 text-4xl">
                      🛋️
                    </div>
                  )}
                </div>

                {/* Card body */}
                <div className="flex flex-col gap-1 p-3 flex-1">
                  <p className="font-medium text-sm leading-tight">{it.name}</p>
                  {it.builderName && (
                    <p className="text-xs text-muted-foreground">by {it.builderName}</p>
                  )}
                  {(it.lastModified ?? it.createdAt) && (() => {
                    const dateStr = it.lastModified ?? it.createdAt
                    const ms = parseLastModified(dateStr)
                    const label = it.lastModified ? "Updated" : "Added"
                    return ms > 0 ? (
                      <p className="text-[10px] text-muted-foreground/60 mt-auto pt-0.5">
                        {label} {relativeTime(ms)}
                      </p>
                    ) : null
                  })()}
                </div>
              </a>
            )
          })}
        </div>
      )}
    </div>
  )
}
