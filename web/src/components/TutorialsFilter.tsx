import { useState, useMemo } from "react"
import { BookOpenIcon, MagnifyingGlassIcon } from "@phosphor-icons/react"
import { BASE } from "@/lib/url"
import { builderSlug } from "@/lib/format"
import { Input } from "@/components/ui/input"

type TagKey = "beginner" | "advanced" | "guild" | "solo" | "sightseeing" | "cn" | "website" | "patch-notes"

interface Tutorial {
  slug: string
  title: string
  description: string
  tags: string[]
  authors: string[]
  image: string | null
  date: Date | null
  updatedDate: Date | null
}

function formatDate(d: Date) {
  return d.toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" })
}

function renderDateLabel(date: Date | null, updatedDate: Date | null): { label: string; updated: boolean } | null {
  if (!date) { return null }
  if (updatedDate && updatedDate.toISOString().slice(0, 10) !== date.toISOString().slice(0, 10)) {
    return { label: formatDate(updatedDate), updated: true }
  }
  return { label: formatDate(date), updated: false }
}

interface Props {
  guides: Tutorial[]
  latestGuides: Tutorial[]
  newestSlug: string
  TAG_CONFIG: Record<TagKey, { label: string; bg: string; text: string; dot: string; ring: string }>
}

type SortKey = "default" | "updated" | "newest"

export default function TutorialsFilter({ guides, latestGuides, newestSlug, TAG_CONFIG }: Props) {
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedTags, setSelectedTags] = useState<Set<TagKey>>(new Set())
  const [sortBy, setSortBy] = useState<SortKey>("default")

  const allTags: TagKey[] = ["beginner", "advanced", "guild", "solo", "sightseeing", "cn", "website", "patch-notes"]

  const filteredGuides = useMemo(() => {
    const q = searchQuery.trim().toLowerCase()
    const filtered = guides.filter((guide) => {
      if (selectedTags.size > 0 && !guide.tags.some((tag) => selectedTags.has(tag as TagKey))) return false
      if (q && !guide.title.toLowerCase().includes(q) && !(guide.description ?? "").toLowerCase().includes(q)) return false
      return true
    })

    if (sortBy === "updated") {
      return [...filtered].sort((a, b) => {
        const ta = (a.updatedDate ?? a.date)?.getTime() ?? 0
        const tb = (b.updatedDate ?? b.date)?.getTime() ?? 0
        return tb - ta
      })
    }
    if (sortBy === "newest") {
      return [...filtered].sort((a, b) => (b.date?.getTime() ?? 0) - (a.date?.getTime() ?? 0))
    }
    return filtered
  }, [guides, selectedTags, sortBy, searchQuery])

  const tagCounts = useMemo(() => {
    const counts: Record<TagKey, number> = {
      beginner: 0,
      advanced: 0,
      guild: 0,
      solo: 0,
      sightseeing: 0,
      cn: 0,
      website: 0,
      "patch-notes": 0,
    }
    guides.forEach((guide) => {
      guide.tags.forEach((tag) => {
        if (tag in counts) {
          counts[tag as TagKey]++
        }
      })
    })
    return counts
  }, [guides])

  const toggleTag = (tag: TagKey) => {
    const newTags = new Set(selectedTags)
    if (newTags.has(tag)) {
      newTags.delete(tag)
    } else {
      newTags.add(tag)
    }
    setSelectedTags(newTags)
  }

  return (
    <div className="space-y-8">
      {latestGuides.length > 0 && (
        <div className="space-y-3">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Recently added</p>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {latestGuides.map((item) => (
              <div key={item.slug} className="group relative flex flex-col rounded-2xl bg-card ring-1 ring-primary/20 overflow-hidden hover:ring-primary/40 hover:shadow-md transition-all">
                <a href={`${BASE}/tutorials/${item.slug}`} className="absolute inset-0" aria-label={item.title} />
                {item.image ? (
                  <div className="aspect-video w-full overflow-hidden bg-muted shrink-0">
                    <img
                      src={item.image.startsWith("http") ? item.image : `${BASE}${item.image}`}
                      alt={item.title}
                      className="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
                      loading="lazy"
                    />
                  </div>
                ) : (
                  <div className="aspect-video w-full bg-primary/5 flex items-center justify-center shrink-0">
                    <BookOpenIcon weight="duotone" className="size-8 text-primary/20" />
                  </div>
                )}
                <div className="relative z-10 flex-1 min-w-0 p-4 space-y-1.5">
                  <div className="flex items-start justify-between gap-2">
                    <a href={`${BASE}/tutorials/${item.slug}`} className="text-sm font-medium leading-snug group-hover:text-primary transition-colors block">
                      {item.title}
                    </a>
                    {item.slug === newestSlug && (
                      <span className="shrink-0 inline-flex items-center rounded-full bg-primary/10 px-2 py-0.5 text-xs font-semibold text-primary">New</span>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground leading-relaxed">{item.description}</p>
                  <div className="flex flex-wrap items-center gap-1.5 pt-0.5">
                    {item.tags
                      .filter((t): t is TagKey => allTags.includes(t as TagKey))
                      .map((t) => {
                        const cfg = TAG_CONFIG[t]
                        return (
                          <span key={t} className={`inline-flex items-center gap-1 rounded-full ${cfg.bg} ${cfg.text} ${cfg.ring} px-2 py-0.5 text-xs font-medium`}>
                            <span className={`size-1.5 rounded-full ${cfg.dot} inline-block`}></span>
                            {cfg.label}
                          </span>
                        )
                      })}
                    {item.authors.length > 0 && (
                      <span className="text-xs text-muted-foreground/70">
                        by {item.authors.map((name, i) => (
                          <span key={`latest-${item.slug}-${name}`}>
                            <a
                              href={`${BASE}/builders/${builderSlug(name)}`}
                              className="relative z-10 hover:text-foreground hover:underline underline-offset-2 transition-colors"
                              onClick={(e) => e.stopPropagation()}
                              data-umami-event="builder_click"
                              data-umami-event-name={name}
                            >
                              {name}
                            </a>
                            {i < item.authors.length - 1 ? ", " : ""}
                          </span>
                        ))}
                      </span>
                    )}
                  </div>
                  {renderDateLabel(item.date, item.updatedDate) && (() => {
                    const dl = renderDateLabel(item.date, item.updatedDate)!
                    return (
                      <p className="text-[11px] text-muted-foreground/60 pt-0.5">
                        {dl.updated ? <span className="text-muted-foreground/80">Updated </span> : null}{dl.label}
                      </p>
                    )
                  })()}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="space-y-4 border-t border-border pt-6">
        <div className="relative">
          <MagnifyingGlassIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-5 text-muted-foreground pointer-events-none" />
          <Input
            type="search"
            placeholder="Search tutorials..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10 h-12 text-base"
            data-umami-event="tutorials_search"
          />
        </div>
        <div className="flex items-center justify-between gap-2 flex-wrap">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">All articles</p>
          <div className="flex items-center gap-1">
            {(["default", "updated", "newest"] as SortKey[]).map((key) => {
              const labels: Record<SortKey, string> = { default: "Chronological", updated: "Recently updated", newest: "Newest" }
              const active = sortBy === key
              return (
                <button
                  key={key}
                  onClick={() => setSortBy(key)}
                  className={`px-2.5 py-1 rounded-full text-xs font-medium transition-all ${
                    active
                      ? "bg-foreground/10 text-foreground"
                      : "text-muted-foreground hover:text-foreground hover:bg-muted/80"
                  }`}
                >
                  {labels[key]}
                </button>
              )
            })}
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {allTags.map((tag) => {
            const cfg = TAG_CONFIG[tag]
            const isSelected = selectedTags.has(tag)
            const count = tagCounts[tag]

            return (
              <button
                key={tag}
                onClick={() => toggleTag(tag)}
                className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-medium transition-all ${
                  isSelected
                    ? `${cfg.bg} ${cfg.text} ${cfg.ring} ring-2`
                    : "bg-muted text-muted-foreground hover:text-foreground hover:bg-muted/80"
                }`}
              >
                <span className={`size-1.5 rounded-full ${isSelected ? cfg.dot : "bg-muted-foreground"} inline-block`}></span>
                {cfg.label}
                {count > 0 && <span className="ml-1 opacity-75">({count})</span>}
              </button>
            )
          })}
        </div>

      {filteredGuides.length === 0 ? (
        <div className="rounded-2xl bg-card ring-1 ring-foreground/10 p-8 text-center">
          <p className="text-muted-foreground">No tutorials match your search.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredGuides.map((item) => (
            <div key={item.slug} className="group relative flex flex-col rounded-2xl bg-card ring-1 ring-foreground/10 overflow-hidden hover:ring-primary/30 hover:shadow-md transition-all">
              <a href={`${BASE}/tutorials/${item.slug}`} className="absolute inset-0" aria-label={item.title} />
              {item.slug === newestSlug && (
                <span className="absolute top-3 right-3 z-10 inline-flex items-center rounded-full bg-primary/10 px-2 py-0.5 text-xs font-semibold text-primary">
                  New
                </span>
              )}
              {item.image ? (
                <div className="aspect-video w-full overflow-hidden bg-muted shrink-0">
                  <img
                    src={item.image.startsWith("http") ? item.image : `${BASE}${item.image}`}
                    alt={item.title}
                    className="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
                    loading="lazy"
                  />
                </div>
              ) : (
                <div className="aspect-video w-full bg-primary/5 flex items-center justify-center shrink-0">
                  <BookOpenIcon weight="duotone" className="size-8 text-primary/20" />
                </div>
              )}
              <div className="relative z-10 flex-1 min-w-0 p-4 space-y-1.5">
                <a href={`${BASE}/tutorials/${item.slug}`} className="text-sm font-medium leading-snug group-hover:text-primary transition-colors block pr-8">
                  {item.title}
                </a>
                <p className="text-xs text-muted-foreground leading-relaxed">{item.description}</p>
                <div className="flex flex-wrap items-center gap-1.5 pt-0.5">
                  {item.tags
                    .filter((t): t is TagKey => allTags.includes(t as TagKey))
                    .map((t) => {
                      const cfg = TAG_CONFIG[t]
                      return (
                        <span key={t} className={`inline-flex items-center gap-1 rounded-full ${cfg.bg} ${cfg.text} ${cfg.ring} px-2 py-0.5 text-xs font-medium`}>
                          <span className={`size-1.5 rounded-full ${cfg.dot} inline-block`}></span>
                          {cfg.label}
                        </span>
                      )
                    })}
                  {item.authors.length > 0 && (
                    <span className="text-xs text-muted-foreground/70">
                      by {item.authors.map((name, i) => (
                        <span key={`${item.slug}-${name}`}>
                          <a
                            href={`${BASE}/builders/${builderSlug(name)}`}
                            className="relative z-10 hover:text-foreground hover:underline underline-offset-2 transition-colors"
                            onClick={(e) => e.stopPropagation()}
                            data-umami-event="builder_click"
                            data-umami-event-name={name}
                          >
                            {name}
                          </a>
                          {i < item.authors.length - 1 ? ", " : ""}
                        </span>
                      ))}
                    </span>
                  )}
                </div>
                {renderDateLabel(item.date, item.updatedDate) && (() => {
                  const dl = renderDateLabel(item.date, item.updatedDate)!
                  return (
                    <p className="text-[11px] text-muted-foreground/60 pt-0.5">
                      {dl.updated ? <span className="text-muted-foreground/80">Updated </span> : null}{dl.label}
                    </p>
                  )
                })()}
              </div>
            </div>
          ))}
        </div>
      )}
      </div>
    </div>
  )
}
