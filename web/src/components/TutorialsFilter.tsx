import { useState, useMemo } from "react"
import { BookOpenIcon } from "@phosphor-icons/react"
import { BASE } from "@/lib/url"

type TagKey = "beginner" | "advanced" | "guild" | "solo" | "sightseeing" | "cn"

interface Tutorial {
  slug: string
  title: string
  description: string
  tags: string[]
  authors: string[]
  image: string | null
}

interface Props {
  guides: Tutorial[]
  newestSlug: string
  TAG_CONFIG: Record<TagKey, { label: string; bg: string; text: string; dot: string; ring: string }>
}

export default function TutorialsFilter({ guides, newestSlug, TAG_CONFIG }: Props) {
  const [selectedTags, setSelectedTags] = useState<Set<TagKey>>(new Set())

  const allTags: TagKey[] = ["beginner", "advanced", "guild", "solo", "sightseeing", "cn"]

  const filteredGuides = useMemo(() => {
    if (selectedTags.size === 0) return guides
    return guides.filter((guide) =>
      guide.tags.some((tag) => selectedTags.has(tag as TagKey))
    )
  }, [guides, selectedTags])

  const tagCounts = useMemo(() => {
    const counts: Record<TagKey, number> = {
      beginner: 0,
      advanced: 0,
      guild: 0,
      solo: 0,
      sightseeing: 0,
      cn: 0,
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
    <div class="space-y-6">
      <div class="flex flex-wrap gap-2">
        {allTags.map((tag) => {
          const cfg = TAG_CONFIG[tag]
          const isSelected = selectedTags.has(tag)
          const count = tagCounts[tag]

          return (
            <button
              key={tag}
              onClick={() => toggleTag(tag)}
              class={`inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-medium transition-all ${
                isSelected
                  ? `${cfg.bg} ${cfg.text} ${cfg.ring} ring-2`
                  : "bg-muted text-muted-foreground hover:text-foreground hover:bg-muted/80"
              }`}
            >
              <span class={`size-1.5 rounded-full ${isSelected ? cfg.dot : "bg-muted-foreground"} inline-block`}></span>
              {cfg.label}
              {count > 0 && <span class="ml-1 opacity-75">({count})</span>}
            </button>
          )
        })}
      </div>

      {filteredGuides.length === 0 ? (
        <div class="rounded-2xl bg-card ring-1 ring-foreground/10 p-8 text-center">
          <p class="text-muted-foreground">No tutorials match the selected tags.</p>
        </div>
      ) : (
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {filteredGuides.map((item) => (
            <div key={item.slug} class="group relative flex flex-col rounded-2xl bg-card ring-1 ring-foreground/10 overflow-hidden hover:ring-primary/30 hover:shadow-md transition-all">
              <a href={`${BASE}/tutorials/${item.slug}`} class="absolute inset-0" aria-label={item.title} />
              {item.slug === newestSlug && (
                <span class="absolute top-3 right-3 z-10 inline-flex items-center rounded-full bg-primary/10 px-2 py-0.5 text-xs font-semibold text-primary">
                  New
                </span>
              )}
              {item.image ? (
                <div class="aspect-video w-full overflow-hidden bg-muted shrink-0">
                  <img
                    src={item.image.startsWith("http") ? item.image : `${BASE}${item.image}`}
                    alt={item.title}
                    class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
                    loading="lazy"
                  />
                </div>
              ) : (
                <div class="aspect-video w-full bg-primary/5 flex items-center justify-center shrink-0">
                  <BookOpenIcon weight="duotone" className="size-8 text-primary/20" />
                </div>
              )}
              <div class="relative z-10 flex-1 min-w-0 p-4 space-y-1.5">
                <a href={`${BASE}/tutorials/${item.slug}`} class="text-sm font-medium leading-snug group-hover:text-primary transition-colors block pr-8">
                  {item.title}
                </a>
                <p class="text-xs text-muted-foreground leading-relaxed">{item.description}</p>
                <div class="flex flex-wrap items-center gap-1.5 pt-0.5">
                  {item.tags
                    .filter((t): t is TagKey => allTags.includes(t as TagKey))
                    .map((t) => {
                      const cfg = TAG_CONFIG[t]
                      return (
                        <span key={t} class={`inline-flex items-center gap-1 rounded-full ${cfg.bg} ${cfg.text} ${cfg.ring} px-2 py-0.5 text-xs font-medium`}>
                          <span class={`size-1.5 rounded-full ${cfg.dot} inline-block`}></span>
                          {cfg.label}
                        </span>
                      )
                    })}
                  {item.authors.length > 0 && (
                    <span class="text-xs text-muted-foreground/70">
                      by {item.authors.map((name, i) => (
                        <span key={`${item.slug}-${name}`}>
                          {name}
                          {i < item.authors.length - 1 ? ", " : ""}
                        </span>
                      ))}
                    </span>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
