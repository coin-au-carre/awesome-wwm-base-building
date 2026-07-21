import * as React from "react"
import { useEffect, useMemo, useState } from "react"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Avatar, CopyPill } from "@/components/GalleryGrid"
import { WBM_RELAY_URL, wbmAvatarsUrl, designerUrl, type DesignerProfile } from "@/lib/gallery"
import { MagnifyingGlassIcon, HammerIcon, HouseIcon, BlueprintIcon, CalculatorIcon, BookOpenIcon, GlobeIcon, ArrowRightIcon, UsersIcon, HeartIcon, StackIcon } from "@phosphor-icons/react"
import { url } from "@/lib/url"

export interface BuilderDirectoryEntry {
  slug: string
  name: string
  discordName?: string
  ingameNickname?: string
  neteaseNumberId?: string
  aliasNames: string[]
  guilds: { name: string; slug: string }[]
  solos: { name: string; slug: string }[]
  tutorials: { title: string; id: string }[]
  guildCount: number
  soloCount: number
  blueprintCount: number
  homesteadSheetCount: number
  tutorialCount: number
  totalCount: number
  isWbmBuilder: boolean
  // Precomputed lowercased blob of every searchable field — built once
  // server-side rather than re-joining these fields on every keystroke.
  searchText: string
}

const SORT_OPTIONS = [
  { value: "total", label: "Total" },
  { value: "guilds", label: "Guild Bases" },
  { value: "solos", label: "Solo Builds" },
  { value: "blueprints", label: "Blueprints" },
  { value: "homestead", label: "Homestead" },
  { value: "tutorials", label: "Tutorials" },
  { value: "name", label: "A–Z" },
] as const

type SortKey = (typeof SORT_OPTIONS)[number]["value"]

function countFor(entry: BuilderDirectoryEntry, sort: SortKey): number {
  switch (sort) {
    case "guilds": return entry.guildCount
    case "solos": return entry.soloCount
    case "blueprints": return entry.blueprintCount
    case "homestead": return entry.homesteadSheetCount
    case "tutorials": return entry.tutorialCount
    default: return entry.totalCount
  }
}

function ContributionBadge({ icon: Icon, count, label, className }: { icon: React.ComponentType<{ weight?: "duotone"; className?: string }>; count: number; label: string; className: string }) {
  if (count === 0) return null
  return (
    <span title={`${count} ${label}${count !== 1 ? "s" : ""}`} className={`inline-flex items-center gap-1 text-[11px] font-medium rounded-full px-2 py-0.5 ${className}`}>
      <Icon weight="duotone" className="size-3" />
      {count}
    </span>
  )
}

function contributionBadges(entry: BuilderDirectoryEntry) {
  return (
    <>
      <ContributionBadge icon={HammerIcon} count={entry.guildCount} label="guild base" className="bg-violet-500/10 text-violet-600 dark:text-violet-300" />
      <ContributionBadge icon={HouseIcon} count={entry.soloCount} label="solo build" className="bg-teal-500/10 text-teal-600 dark:text-teal-300" />
      <ContributionBadge icon={BlueprintIcon} count={entry.blueprintCount} label="blueprint" className="bg-blue-500/10 text-blue-600 dark:text-blue-300" />
      <ContributionBadge icon={CalculatorIcon} count={entry.homesteadSheetCount} label="homestead sheet" className="bg-orange-500/10 text-orange-600 dark:text-orange-300" />
      <ContributionBadge icon={BookOpenIcon} count={entry.tutorialCount} label="tutorial" className="bg-pink-500/10 text-pink-600 dark:text-pink-300" />
      {entry.totalCount === 0 && <Badge variant="outline" className="text-[11px]">No contributions yet</Badge>}
    </>
  )
}

// One inline stat — icon + count + label, several sit in a row rather
// than each getting its own boxed column (which looked heavy for just
// three numbers).
function InlineStat({ icon: Icon, value, label, className }: { icon: React.ComponentType<{ weight?: "fill"; className?: string }>; value: number; label: string; className: string }) {
  return (
    <span className="flex items-center gap-1.5 text-sm" title={label}>
      <Icon weight="fill" className={`size-4 ${className}`} />
      <span className="font-semibold">{value.toLocaleString()}</span>
      <span className="text-muted-foreground">{label}</span>
    </span>
  )
}

// Right-column detail panel for the selected builder — avatarUrl comes
// from the one bulk /api/gallery/wbm/avatars fetch in BuildersDirectory,
// not a per-selection request. Fans/Likes/Published Works are only
// fetched here, on selection, for WBM builders specifically — the bulk
// avatars endpoint doesn't carry those (it only resolves what
// pidToAuthorInfo already needs for author cards elsewhere), and fetching
// them for the whole roster up front would be one designerUrl call per
// WBM builder for stats nobody may ever look at.
function BuilderDetailPanel({ entry, avatarUrl }: { entry: BuilderDirectoryEntry; avatarUrl?: string }) {
  const initial = entry.name.trim()[0]?.toUpperCase() ?? "?"
  const [profile, setProfile] = useState<DesignerProfile | null>(null)

  useEffect(() => {
    setProfile(null)
    if (!entry.isWbmBuilder || !entry.neteaseNumberId || !WBM_RELAY_URL) return
    fetch(designerUrl(entry.neteaseNumberId))
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => data && setProfile(data))
      .catch(() => {})
  }, [entry.isWbmBuilder, entry.neteaseNumberId])

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4">
        {avatarUrl ? (
          <Avatar src={avatarUrl} className="flex size-24" />
        ) : (
          <div className="flex size-24 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary text-3xl font-bold ring-2 ring-primary/20">
            {initial}
          </div>
        )}
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="font-heading text-xl font-bold truncate">{entry.name}</h2>
            {entry.isWbmBuilder && (
              <img src={url("/images/logo_1.webp")} alt="WBM Builder" title="WBM Builder" className="size-6 object-contain shrink-0" />
            )}
          </div>
          {(entry.discordName || entry.ingameNickname || entry.aliasNames.length > 0) && (
            <p className="text-xs text-muted-foreground">
              {[
                entry.discordName && `Discord: ${entry.discordName}`,
                entry.ingameNickname && entry.ingameNickname !== entry.discordName && `In-game: ${entry.ingameNickname}`,
                entry.aliasNames.length > 0 && `also known as ${entry.aliasNames.join(", ")}`,
              ].filter(Boolean).join(" · ")}
            </p>
          )}
        </div>
      </div>

      {entry.neteaseNumberId && <CopyPill label="Character ID" value={entry.neteaseNumberId} />}

      {profile && (
        <div className="flex flex-wrap items-center gap-4 text-xs">
          <InlineStat icon={UsersIcon} value={profile.follower_num} label="Fans" className="text-blue-400" />
          <InlineStat icon={HeartIcon} value={profile.like_num} label="Likes" className="text-rose-400" />
          <InlineStat icon={StackIcon} value={profile.published_num} label="Published Works" className="text-amber-400" />
        </div>
      )}

      <div className="flex flex-wrap items-center gap-1.5">{contributionBadges(entry)}</div>

      {entry.guilds.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-1.5">Guild Bases</h3>
          <div className="flex flex-wrap gap-1.5">
            {entry.guilds.map((g) => (
              <a
                key={g.slug}
                href={url(`/guilds/${g.slug}`)}
                className="text-xs rounded-full px-2.5 py-1 bg-violet-500/10 text-violet-600 dark:text-violet-300 ring-1 ring-inset ring-violet-500/25 hover:ring-violet-500/50 transition-colors"
              >
                {g.name}
              </a>
            ))}
          </div>
        </div>
      )}

      {entry.solos.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-1.5">Solo Bases</h3>
          <div className="flex flex-wrap gap-1.5">
            {entry.solos.map((s) => (
              <a
                key={s.slug}
                href={url(`/solos/${s.slug}`)}
                className="text-xs rounded-full px-2.5 py-1 bg-teal-500/10 text-teal-600 dark:text-teal-300 ring-1 ring-inset ring-teal-500/25 hover:ring-teal-500/50 transition-colors"
              >
                {s.name}
              </a>
            ))}
          </div>
        </div>
      )}

      {entry.tutorials.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-1.5">Tutorials</h3>
          <div className="flex flex-col gap-1.5">
            {entry.tutorials.map((t) => (
              <a
                key={t.id}
                href={url(`/tutorials/${t.id}`)}
                className="group flex items-center gap-2 rounded-lg ring-1 ring-pink-500/25 bg-pink-500/5 hover:ring-pink-500/50 hover:bg-pink-500/10 transition-all px-3 py-2"
              >
                <BookOpenIcon weight="duotone" className="size-4 text-pink-500 shrink-0" />
                <span className="text-sm truncate group-hover:text-pink-600 dark:group-hover:text-pink-300 transition-colors">{t.title}</span>
              </a>
            ))}
          </div>
        </div>
      )}

      <a
        href={url(`/builders/${entry.slug}`)}
        className="inline-flex items-center gap-1.5 text-sm font-medium text-primary hover:underline"
      >
        View full profile <ArrowRightIcon weight="bold" className="size-3.5" />
      </a>
    </div>
  )
}

export function BuildersDirectory({ entries }: { entries: BuilderDirectoryEntry[] }) {
  const [query, setQuery] = useState("")
  const [sort, setSort] = useState<SortKey>("total")
  const [wbmOnly, setWbmOnly] = useState(true)
  const [selectedSlug, setSelectedSlug] = useState<string | null>(null)
  // number_id -> avatar_url for every WBM builder, one bulk request
  // instead of one designerUrl call per row/selection (see wbmAvatarsUrl's
  // doc comment).
  const [avatars, setAvatars] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!WBM_RELAY_URL) return
    fetch(wbmAvatarsUrl())
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => data && setAvatars(data))
      .catch(() => {})
  }, [])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    let list = wbmOnly ? entries.filter((e) => e.isWbmBuilder) : entries
    if (q) list = list.filter((e) => e.searchText.includes(q))
    return [...list].sort((a, b) => {
      if (sort === "name") return a.name.localeCompare(b.name, undefined, { sensitivity: "base" })
      const diff = countFor(b, sort) - countFor(a, sort)
      return diff !== 0 ? diff : a.name.localeCompare(b.name, undefined, { sensitivity: "base" })
    })
  }, [entries, query, sort, wbmOnly])

  const selected = useMemo(
    () => (selectedSlug ? (filtered.find((e) => e.slug === selectedSlug) ?? entries.find((e) => e.slug === selectedSlug)) : undefined),
    [selectedSlug, filtered, entries],
  )

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <Tabs value={wbmOnly ? "wbm" : "all"} onValueChange={(v) => setWbmOnly(v === "wbm")}>
          <TabsList>
            <TabsTrigger value="wbm" className="inline-flex items-center gap-1.5">
              <img src={url("/images/logo_1.webp")} alt="" aria-hidden="true" className="size-5 object-contain" />
              WBM Builders
            </TabsTrigger>
            <TabsTrigger value="all" className="inline-flex items-center gap-1.5">
              <GlobeIcon weight="bold" className="size-4" />
              Referenced Builders
            </TabsTrigger>
          </TabsList>
        </Tabs>
        <div className="relative flex-1 min-w-56">
          <MagnifyingGlassIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search by Discord name, in-game name, alias, or Character ID…"
            className="pl-9"
          />
        </div>
        <Tabs value={sort} onValueChange={(v) => setSort(v as SortKey)}>
          <TabsList className="flex-wrap h-auto">
            {SORT_OPTIONS.map((opt) => (
              <TabsTrigger key={opt.value} value={opt.value}>{opt.label}</TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      <p className="text-xs text-muted-foreground">
        {filtered.length} builder{filtered.length !== 1 ? "s" : ""}{query && ` matching "${query}"`}
      </p>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 items-start">
        {filtered.length === 0 ? (
          <p className="text-sm text-muted-foreground py-8 text-center lg:col-span-2">No builders match your search.</p>
        ) : (
          <div className="rounded-xl ring-1 ring-border divide-y divide-border overflow-hidden max-h-[70vh] overflow-y-auto">
            {filtered.map((entry) => (
              <button
                key={entry.slug}
                type="button"
                onClick={() => setSelectedSlug(entry.slug)}
                className={`w-full flex flex-wrap items-center gap-3 px-4 py-3 text-left transition-colors cursor-pointer ${entry.slug === selectedSlug ? "bg-primary/10" : "hover:bg-muted/50"}`}
              >
                <Avatar
                  src={entry.neteaseNumberId ? avatars[entry.neteaseNumberId] : undefined}
                  className="flex size-9 shrink-0"
                />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium truncate">{entry.name}</span>
                    {entry.isWbmBuilder && (
                      <img src={url("/images/logo_1.webp")} alt="WBM Builder" title="WBM Builder" className="size-5 object-contain shrink-0" />
                    )}
                  </div>
                  {(entry.discordName || entry.ingameNickname || entry.aliasNames.length > 0) && (
                    <p className="text-xs text-muted-foreground truncate">
                      {[
                        entry.discordName && `Discord: ${entry.discordName}`,
                        entry.ingameNickname && entry.ingameNickname !== entry.discordName && `In-game: ${entry.ingameNickname}`,
                        entry.aliasNames.length > 0 && `also known as ${entry.aliasNames.join(", ")}`,
                      ].filter(Boolean).join(" · ")}
                    </p>
                  )}
                </div>
                <div className="flex flex-wrap items-center gap-1.5 shrink-0">{contributionBadges(entry)}</div>
              </button>
            ))}
          </div>
        )}

        <div className="rounded-xl ring-1 ring-border bg-card p-5 lg:sticky lg:top-4">
          {selected ? (
            <BuilderDetailPanel entry={selected} avatarUrl={selected.neteaseNumberId ? avatars[selected.neteaseNumberId] : undefined} />
          ) : (
            <p className="text-sm text-muted-foreground text-center py-8">Select a builder from the list to see their details.</p>
          )}
        </div>
      </div>
    </div>
  )
}
