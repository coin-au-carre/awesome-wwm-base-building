import { useState, useMemo } from "react"
import * as React from "react"
import type { RankedBlueprint } from "@/types/blueprint"
import { url } from "@/lib/url"
import { builderSlug } from "@/lib/format"
import { cn } from "@/lib/utils"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetTrigger,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetFooter,
} from "@/components/ui/sheet"
import { Search, X, SlidersHorizontal } from "lucide-react"

declare global {
  interface Window {
    umami?: { track: (event: string, data?: Record<string, unknown>) => void }
  }
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

type PriceFilter = "all" | "free" | "paytobuild"

interface Props {
  blueprints: RankedBlueprint[]
  allTags: string[]
}

export function BlueprintGrid({ blueprints, allTags }: Props) {
  const [search, setSearch] = useState("")
  const [activeTags, setActiveTags] = useState<Set<string>>(new Set())
  const [priceFilter, setPriceFilter] = useState<PriceFilter>("all")
  const [sheetOpen, setSheetOpen] = useState(false)

  function toggleTag(tag: string) {
    setActiveTags((prev) => {
      const next = new Set(prev)
      if (next.has(tag)) next.delete(tag)
      else next.add(tag)
      return next
    })
  }

  const filtered = useMemo(() => {
    let result = blueprints
    if (search.trim()) {
      const q = search.toLowerCase()
      result = result.filter(
        (bp) =>
          bp.name.toLowerCase().includes(q) ||
          bp.builderName?.toLowerCase().includes(q) ||
          bp.materials?.toLowerCase().includes(q)
      )
    }
    if (activeTags.size > 0) {
      result = result.filter((bp) => bp.tags?.some((t) => activeTags.has(t)))
    }
    if (priceFilter === "free") {
      result = result.filter((bp) => bp.isFree === true)
    } else if (priceFilter === "paytobuild") {
      result = result.filter((bp) => bp.isPayToBuild === true)
    }
    return result
  }, [blueprints, search, activeTags, priceFilter])

  const activeFilterCount = activeTags.size + (priceFilter !== "all" ? 1 : 0)

  const TagFilters = (
    <div className="flex flex-wrap gap-2">
      {allTags.map((tag) => (
        <Tag key={tag} label={tag} active={activeTags.has(tag)} onClick={() => toggleTag(tag)} />
      ))}
    </div>
  )

  return (
    <div className="space-y-4">
      {/* Search + filter bar */}
      <div className="flex gap-2 items-center">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search blueprints…"
            className="pl-8 pr-8"
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

        {/* Price filter pills — desktop */}
        <div className="hidden sm:flex gap-1">
          {(["all", "free", "paytobuild"] as PriceFilter[]).map((f) => (
            <button
              key={f}
              type="button"
              onClick={() => setPriceFilter(f)}
              className={cn(
                "rounded-full px-3 py-1 text-xs font-medium transition-colors border",
                priceFilter === f
                  ? "bg-foreground text-background border-foreground"
                  : "bg-transparent text-muted-foreground border-border hover:text-foreground"
              )}
            >
              {f === "all" ? "All" : f === "free" ? "Free" : "Pay-to-build"}
            </button>
          ))}
        </div>

        {/* Mobile: sheet */}
        <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
          <SheetTrigger asChild>
            <Button variant="outline" size="sm" className="sm:hidden shrink-0 gap-1.5">
              <SlidersHorizontal className="size-3.5" />
              Filters{activeFilterCount > 0 ? ` (${activeFilterCount})` : ""}
            </Button>
          </SheetTrigger>
          <SheetContent side="bottom" className="max-h-[70dvh] overflow-y-auto rounded-t-2xl">
            <SheetHeader className="mb-4">
              <SheetTitle>Filter blueprints</SheetTitle>
            </SheetHeader>
            <div className="space-y-5 pb-2">
              <div>
                <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2">Price</p>
                <div className="flex flex-wrap gap-2">
                  {(["all", "free", "paytobuild"] as PriceFilter[]).map((f) => (
                    <button
                      key={f}
                      type="button"
                      onClick={() => setPriceFilter(f)}
                      className={cn(
                        "rounded-full px-3 py-1 text-xs font-medium transition-colors border",
                        priceFilter === f
                          ? "bg-foreground text-background border-foreground"
                          : "bg-transparent text-muted-foreground border-border hover:text-foreground"
                      )}
                    >
                      {f === "all" ? "All" : f === "free" ? "Free" : "Pay-to-build"}
                    </button>
                  ))}
                </div>
              </div>
              {allTags.length > 0 && (
                <div>
                  <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2">Tags</p>
                  {TagFilters}
                </div>
              )}
            </div>
            <SheetFooter className="mt-4">
              {activeFilterCount > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => { setActiveTags(new Set()); setPriceFilter("all") }}
                >
                  Clear all
                </Button>
              )}
              <Button onClick={() => setSheetOpen(false)}>Done</Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Desktop: tag filter (inline Sheet-like) trigger */}
        {allTags.length > 0 && (
          <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
            <SheetTrigger asChild>
              <Button variant="outline" size="sm" className="hidden sm:flex shrink-0 gap-1.5">
                <SlidersHorizontal className="size-3.5" />
                Tags{activeTags.size > 0 ? ` (${activeTags.size})` : ""}
              </Button>
            </SheetTrigger>
            <SheetContent side="bottom" className="max-h-[60dvh] overflow-y-auto rounded-t-2xl">
              <SheetHeader className="mb-4">
                <SheetTitle>Filter by tag</SheetTitle>
              </SheetHeader>
              {TagFilters}
              <SheetFooter className="mt-4">
                {activeTags.size > 0 && (
                  <Button variant="ghost" size="sm" onClick={() => setActiveTags(new Set())}>
                    Clear tags
                  </Button>
                )}
                <Button onClick={() => setSheetOpen(false)}>Done</Button>
              </SheetFooter>
            </SheetContent>
          </Sheet>
        )}
      </div>

      {/* Results count */}
      {(search || activeFilterCount > 0) && (
        <p className="text-sm text-muted-foreground">
          {filtered.length} blueprint{filtered.length !== 1 ? "s" : ""}
          {activeFilterCount > 0 && (
            <button
              type="button"
              onClick={() => { setActiveTags(new Set()); setPriceFilter("all") }}
              className="ml-2 text-xs underline hover:text-foreground"
            >
              Clear filters
            </button>
          )}
        </p>
      )}

      {/* Grid */}
      {filtered.length === 0 ? (
        <p className="text-sm text-muted-foreground py-8 text-center">No blueprints match your filters.</p>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((bp) => {
            const img = bp.coverImage ?? bp.screenshots?.[0]
            return (
              <a
                key={bp.discordThread}
                href={url(`/blueprints/${bp.slug}`)}
                className="group relative flex flex-col overflow-hidden rounded-xl ring-1 ring-border bg-card hover:ring-primary transition-all"
                data-umami-event="blueprint_click"
                data-umami-event-name={bp.name}
              >
                {/* Thumbnail */}
                <div className="relative aspect-video bg-muted overflow-hidden">
                  {img ? (
                    <img
                      src={img}
                      alt={bp.name}
                      loading="lazy"
                      onerror="this.style.display='none'"
                      className="absolute inset-0 h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                    />
                  ) : (
                    <div className="absolute inset-0 flex items-center justify-center text-muted-foreground/30 text-4xl">
                      📐
                    </div>
                  )}
                  {/* Price badge */}
                  <div className="absolute top-2 right-2">
                    {bp.isFree && bp.isPayToBuild ? (
                      <span className="inline-flex items-center rounded-full bg-sky-600/90 px-2 py-0.5 text-[10px] font-semibold text-white backdrop-blur-sm">
                        Free + Paid
                      </span>
                    ) : bp.isPayToBuild ? (
                      <span className="inline-flex items-center rounded-full bg-amber-500/90 px-2 py-0.5 text-[10px] font-semibold text-white backdrop-blur-sm">
                        Pay-to-build
                      </span>
                    ) : (
                      <span className="inline-flex items-center rounded-full bg-emerald-600/90 px-2 py-0.5 text-[10px] font-semibold text-white backdrop-blur-sm">
                        Free
                      </span>
                    )}
                  </div>
                </div>

                {/* Card body */}
                <div className="flex flex-col gap-1.5 p-3 flex-1">
                  <div className="flex items-start justify-between gap-2">
                    <p className="font-medium text-sm leading-tight">{bp.name}</p>
                    {bp.score > 0 && (
                      <span className="shrink-0 text-xs text-muted-foreground">⭐ {bp.score}</span>
                    )}
                  </div>
                  {bp.builderName && (
                    <p className="text-xs text-muted-foreground">by <a href={url(`/builders/${builderSlug(bp.builderName)}`)} className="hover:text-foreground hover:underline underline-offset-2 transition-colors" onClick={(e) => e.stopPropagation()}>{bp.builderName}</a></p>
                  )}
                  {bp.price && bp.isPayToBuild && (
                    <p className="text-xs text-amber-600 dark:text-amber-400">{bp.price}</p>
                  )}
                  {bp.materials && (
                    <p className="text-xs text-muted-foreground line-clamp-1">
                      Materials: {bp.materials}
                    </p>
                  )}
                  {bp.tags && bp.tags.length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-auto pt-1">
                      {bp.tags.slice(0, 3).map((tag) => {
                        const cfg = tagPalette(tag)
                        return (
                          <span
                            key={tag}
                            className={cn("inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium", cfg.base)}
                          >
                            {tag}
                          </span>
                        )
                      })}
                    </div>
                  )}
                </div>
              </a>
            )
          })}
        </div>
      )}
    </div>
  )
}
