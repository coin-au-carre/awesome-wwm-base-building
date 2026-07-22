import * as React from "react"
import { useEffect, useMemo, useRef, useState } from "react"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuCheckboxItem } from "@/components/ui/dropdown-menu"
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { AvatarStatus, CopyPill } from "@/components/GalleryGrid"
import { WBM_RELAY_URL, wbmAvatarsUrl, wbmStatusUrl, designerUrl, deviceLabel, type DesignerProfile, type BuilderStatus } from "@/lib/gallery"
import { MagnifyingGlassIcon, HammerIcon, HouseIcon, BlueprintIcon, CalculatorIcon, BookOpenIcon, GlobeIcon, ArrowRightIcon, UsersIcon, HeartIcon, StackIcon, SwordIcon, CaretDownIcon } from "@phosphor-icons/react"
import { url } from "@/lib/url"
import { BuilderExtraInfo } from "@/components/BuilderExtraInfo"
import { BuilderMedia } from "@/components/BuilderMedia"
import type { Channel, TutorialVideo } from "@/lib/videos"

// Matches Tailwind's `lg` breakpoint (1024px) — the same one the grid
// layout below switches on. Scrolling a long list just to reach the
// detail panel is painful on mobile (no sticky side column to keep it
// in view), so below this width the detail opens in a Sheet instead of
// inline in the page flow; at/above it, nothing changes from before.
function useIsDesktop(): boolean {
  const [isDesktop, setIsDesktop] = useState(() => typeof window !== "undefined" && window.matchMedia("(min-width: 1024px)").matches)
  useEffect(() => {
    const mql = window.matchMedia("(min-width: 1024px)")
    const onChange = () => setIsDesktop(mql.matches)
    mql.addEventListener("change", onChange)
    return () => mql.removeEventListener("change", onChange)
  }, [])
  return isDesktop
}

// Generic multi-select dropdown (checkboxes) — used for the Region/
// Platform filters. Shows "All" when nothing's selected, the single
// value when exactly one is picked, or a count otherwise, so the
// trigger stays compact regardless of how many options exist.
// Fixed display order for the platform dropdown (raw device_name
// values, not the friendly labels) — unrecognized values sort after
// these, alphabetically among themselves.
const DEVICE_ORDER = ["windows", "ios", "android", "prospero"]

function deviceSortCompare(a: string, b: string): number {
  const ai = DEVICE_ORDER.indexOf(a.toLowerCase())
  const bi = DEVICE_ORDER.indexOf(b.toLowerCase())
  if (ai === -1 && bi === -1) {return a.localeCompare(b)}
  if (ai === -1) {return 1}
  if (bi === -1) {return -1}
  return ai - bi
}

function MultiSelectFilter({
  label,
  options,
  selected,
  onChange,
  formatLabel = (opt) => opt,
}: {
  label: string
  options: string[]
  selected: string[]
  onChange: (next: string[]) => void
  // Raw values (e.g. device_name's "prospero") stay the filter/toggle
  // key — this only affects what's shown, so PS5/Xbox-style friendly
  // names don't have to leak into filtering logic.
  formatLabel?: (opt: string) => string
}) {
  function toggle(opt: string) {
    onChange(selected.includes(opt) ? selected.filter((o) => o !== opt) : [...selected, opt])
  }

  const summary = selected.length === 0 ? `All ${label}s` : selected.length === 1 ? formatLabel(selected[0]) : `${selected.length} ${label}s`

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className="gap-1.5">
          {summary}
          <CaretDownIcon weight="bold" className="size-3.5 text-muted-foreground" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        {options.map((opt) => (
          <DropdownMenuCheckboxItem key={opt} checked={selected.includes(opt)} onCheckedChange={() => toggle(opt)} onSelect={(e) => e.preventDefault()}>
            {formatLabel(opt)}
          </DropdownMenuCheckboxItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

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
  // This builder's own channel(s)/video(s) from lib/videos.ts (see
  // lib/builders.ts's getBuilderProfile) — rendered via BuilderMedia,
  // same component /builders/[slug] uses.
  channels: Channel[]
  tutorialVideos: TutorialVideo[]
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
  { value: "tutorials", label: "Tutorials" },
  { value: "martial_mastery", label: "Martial Mastery" },
  { value: "name", label: "A–Z" },
] as const

type SortKey = (typeof SORT_OPTIONS)[number]["value"]

// martial_mastery needs statuses (keyed by number_id) since that stat
// lives in the bulk status fetch, not on BuilderDirectoryEntry itself —
// every other sort key is a plain field on entry.
function countFor(entry: BuilderDirectoryEntry, sort: SortKey, statuses: Record<string, BuilderStatus>): number {
  switch (sort) {
    case "tutorials": return entry.tutorialCount
    case "martial_mastery": return (entry.neteaseNumberId && statuses[entry.neteaseNumberId]?.max_xiuwei_kungfu) || 0
    default: return entry.totalCount
  }
}

// Row-list badges are hidden below `sm` entirely now (see the list
// rendering below) — the row only needs to stay readable at `sm`+, so
// this can just always show its full label text.
function ContributionBadge({ icon: Icon, count, label, className }: { icon: React.ComponentType<{ weight?: "duotone"; className?: string }>; count: number; label: string; className: string }) {
  if (count === 0) { return null }
  const plural = `${label}${count !== 1 ? "s" : ""}`
  const display = plural.charAt(0).toUpperCase() + plural.slice(1)
  return (
    <span className={`inline-flex items-center gap-1 text-[11px] font-medium rounded-full px-2 py-0.5 ${className}`}>
      <Icon weight="duotone" className="size-3" />
      {count} {display}
    </span>
  )
}

// Plain-text tag pill for region/device — same rounded-pill language as
// ContributionBadge, but no icon/count, just a label.
function Tag({ children, className, title }: { children: React.ReactNode; className: string; title?: string }) {
  return <span title={title} className={`inline-flex items-center gap-1 text-[11px] font-medium rounded-full px-2 py-0.5 ${className}`}>{children}</span>
}

function statusTags(status: BuilderStatus | undefined) {
  if (!status) { return null }
  return (
    <>
      {status.oversea_tag && <Tag className="bg-sky-500/10 text-sky-600 dark:text-sky-300">{status.oversea_tag}</Tag>}
      {status.device_name && <Tag className="bg-slate-500/10 text-slate-600 dark:text-slate-300">{deviceLabel(status.device_name)}</Tag>}
      {!!status.max_xiuwei_kungfu && (
        <Tag className="bg-amber-500/10 text-white" title="Martial Mastery">
          <SwordIcon weight="fill" className="size-3" />
          {status.max_xiuwei_kungfu.toLocaleString()}
        </Tag>
      )}
    </>
  )
}

function contributionBadges(entry: BuilderDirectoryEntry) {
  return (
    <>
      <ContributionBadge icon={HammerIcon} count={entry.guildCount} label="guild" className="bg-violet-500/10 text-violet-600 dark:text-violet-300"/>
      <ContributionBadge icon={HouseIcon} count={entry.soloCount} label="solo" className="bg-teal-500/10 text-teal-600 dark:text-teal-300"/>
      <ContributionBadge icon={BlueprintIcon} count={entry.blueprintCount} label="blueprint" className="bg-blue-500/10 text-blue-600 dark:text-blue-300"/>
      <ContributionBadge icon={CalculatorIcon} count={entry.homesteadSheetCount} label="homestead sheet" className="bg-orange-500/10 text-orange-600 dark:text-orange-300" />
      <ContributionBadge icon={BookOpenIcon} count={entry.tutorialCount} label="tuto" className="bg-pink-500/10 text-pink-600 dark:text-pink-300" />
      {entry.totalCount === 0 && <Badge variant="outline" className="text-[11px]">No WBM contributions yet</Badge>}
    </>
  )
}

// One inline stat — icon + count + label, several sit in a row rather
// than each getting its own boxed column (which looked heavy for just
// three numbers).
export function InlineStat({ icon: Icon, value, label, className }: { icon: React.ComponentType<{ weight?: "fill"; className?: string }>; value: number; label: string; className: string }) {
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
function BuilderDetailPanel({ entry, avatarUrl, status }: { entry: BuilderDirectoryEntry; avatarUrl?: string; status?: BuilderStatus }) {
  const initial = entry.name.trim()[0]?.toUpperCase() ?? "?"
  const [profile, setProfile] = useState<DesignerProfile | null>(null)

  useEffect(() => {
    if (!entry.isWbmBuilder || !entry.neteaseNumberId || !WBM_RELAY_URL) {return}
    fetch(designerUrl(entry.neteaseNumberId))
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => data && setProfile(data))
      .catch(() => {})
  }, [entry.isWbmBuilder, entry.neteaseNumberId])

  // Discord name / in-game nickname / aliases collapsed into one line
  // (same pattern as the full profile page's identityLine) instead of up
  // to three stacked <p>s, so the header block stays short and the panel
  // gets more of its height budget below the fold.
  const identityLine = [
    entry.discordName && `Discord: ${entry.discordName}`,
    entry.ingameNickname && entry.ingameNickname !== entry.discordName ? `In-game: ${entry.ingameNickname}` : null,
    entry.aliasNames.length > 0 ? `also known as ${entry.aliasNames.join(", ")}` : null,
  ].filter(Boolean).join(" · ")

  return (
    <div className="space-y-3">
      <div className="flex items-start gap-4">
        <div style={{ viewTransitionName: `builder-avatar-${entry.slug}` }}>
          {avatarUrl ? (
            <AvatarStatus src={avatarUrl} className="flex size-24" level={status?.level} isOnline={status?.is_online} />
          ) : (
            <div className="flex size-24 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary text-3xl font-bold ring-2 ring-primary/20">
              {initial}
            </div>
          )}
        </div>
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h2 style={{ viewTransitionName: `builder-name-${entry.slug}` }} className="font-heading text-xl font-bold truncate">{entry.name}</h2>
            {!!status?.level && <span className="font-heading text-lg font-bold text-primary shrink-0">Lv.{status.level}</span>}
            {entry.isWbmBuilder && (
              <img src={url("/images/logo_1.webp")} alt="WBM Builder" title="WBM Builder" className="size-6 object-contain shrink-0" />
            )}
          </div>
          {identityLine && <p className="text-xs text-muted-foreground truncate">{identityLine}</p>}
          <div className="flex flex-wrap items-center gap-3 mt-1.5">
            {entry.neteaseNumberId && <CopyPill label="Character ID" value={entry.neteaseNumberId} />}
            {profile && (
              <>
                <InlineStat icon={UsersIcon} value={profile.follower_num} label="Fans" className="text-blue-400" />
                <InlineStat icon={HeartIcon} value={profile.like_num} label="Likes" className="text-rose-400" />
                <InlineStat icon={StackIcon} value={profile.published_num} label="Published Works" className="text-amber-400" />
              </>
            )}
          </div>
        </div>
      </div>

      {profile && <BuilderExtraInfo profile={profile} compact />}

      <div className="flex flex-wrap items-center gap-1.5">
        {contributionBadges(entry)}
      </div>

      <BuilderMedia channels={entry.channels} />

      {(entry.guilds.length > 0 || entry.solos.length > 0) && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
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
        </div>
      )}

      {entry.tutorials.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-1.5">Tutorials</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-1.5">
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
        className={`${buttonVariants({ variant: "default" })} w-full`}
      >
        View full profile <ArrowRightIcon weight="bold" className="size-3" />
      </a>
    </div>
  )
}

// Reads the directory's filter/selection state out of the current URL's
// query string — used for lazy useState initializers below, so a page
// load (including a browser back-navigation landing back on /builders)
// restores synchronously on first render rather than flashing default
// state before an effect catches up. See the mirroring writer effect
// (syncs state -> URL via history.replaceState) further down.
function readDirectoryParams(): URLSearchParams {
  if (typeof window === "undefined") {return new URLSearchParams()}
  return new URLSearchParams(window.location.search)
}

export function BuildersDirectory({ entries }: { entries: BuilderDirectoryEntry[] }) {
  const [query, setQuery] = useState(() => readDirectoryParams().get("q") ?? "")
  const [sort, setSort] = useState<SortKey>(() => (readDirectoryParams().get("sort") as SortKey | null) ?? "total")
  const [wbmOnly, setWbmOnly] = useState(() => readDirectoryParams().get("wbm") !== "0")
  const [selectedSlug, setSelectedSlug] = useState<string | null>(() => readDirectoryParams().get("sel"))
  // Scrolled into view on selection — matters most on mobile/narrow
  // screens where the detail panel stacks below a possibly long list
  // (no more sticky side panel to keep it in view automatically). Guarded
  // by hasMountedRef so restoring selectedSlug from the URL on first
  // render doesn't also trigger a scroll-jump — only a later, genuine
  // click should animate.
  const detailRef = useRef<HTMLDivElement>(null)
  const hasMountedRef = useRef(false)
  // number_id -> avatar_url for every WBM builder, one bulk request
  // instead of one designerUrl call per row/selection (see wbmAvatarsUrl's
  // doc comment).
  const [avatars, setAvatars] = useState<Record<string, string>>({})
  // number_id -> level/online status, same bulk pattern but its own
  // short-TTL endpoint (see wbmStatusUrl's doc comment) since status
  // goes stale far faster than avatars/names.
  const [statuses, setStatuses] = useState<Record<string, BuilderStatus>>({})
  const [regionFilter, setRegionFilter] = useState<string[]>(() => {
    const v = readDirectoryParams().get("region")
    return v ? v.split(",") : []
  })
  const [deviceFilter, setDeviceFilter] = useState<string[]>(() => {
    const v = readDirectoryParams().get("device")
    return v ? v.split(",") : []
  })
  const [onlineOnly, setOnlineOnly] = useState(() => readDirectoryParams().get("online") === "1")

  // Mirrors filter/selection state into the URL (replaceState — doesn't
  // push a new history entry per keystroke/toggle, just keeps the
  // current one in sync) so that navigating to a builder's full profile
  // and then back restores the exact same view. Omits anything at its
  // default so the URL stays clean when nothing's been touched.
  useEffect(() => {
    const params = new URLSearchParams()
    if (query) {params.set("q", query)}
    if (sort !== "total") {params.set("sort", sort)}
    if (!wbmOnly) {params.set("wbm", "0")}
    if (regionFilter.length > 0) {params.set("region", regionFilter.join(","))}
    if (deviceFilter.length > 0) {params.set("device", deviceFilter.join(","))}
    if (onlineOnly) {params.set("online", "1")}
    if (selectedSlug) {params.set("sel", selectedSlug)}
    const qs = params.toString()
    const newUrl = qs ? `${window.location.pathname}?${qs}` : window.location.pathname
    window.history.replaceState(window.history.state, "", newUrl)
  }, [query, sort, wbmOnly, regionFilter, deviceFilter, onlineOnly, selectedSlug])

  useEffect(() => {
    if (!WBM_RELAY_URL) {return }
    fetch(wbmAvatarsUrl())
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => data && setAvatars(data))
      .catch(() => {})
    fetch(wbmStatusUrl())
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => data && setStatuses(data))
      .catch(() => {})
  }, [])

  // Region/device options are derived from whatever the live status
  // fetch actually returned, not a hardcoded list — avoids listing a
  // region/platform nobody on the roster actually has, and needs no
  // maintenance if NetEase adds a new one.
  const regionOptions = useMemo(
    () => Array.from(new Set(Object.values(statuses).map((s) => s.oversea_tag).filter((v): v is string => !!v))).sort(),
    [statuses],
  )
  const deviceOptions = useMemo(
    () => Array.from(new Set(Object.values(statuses).map((s) => s.device_name).filter((v): v is string => !!v))).sort(deviceSortCompare),
    [statuses],
  )

  // Everything except the online-only toggle — computed separately so
  // "X/Y online" (below) reflects search/region/device filters without
  // itself being collapsed to "Y/Y" once online-only is switched on.
  const beforeOnlineFilter = useMemo(() => {
    const q = query.trim().toLowerCase()
    let list = wbmOnly ? entries.filter((e) => e.isWbmBuilder) : entries
    if (q) {list = list.filter((e) => e.searchText.includes(q))}
    if (regionFilter.length > 0) {list = list.filter((e) => e.neteaseNumberId && regionFilter.includes(statuses[e.neteaseNumberId]?.oversea_tag ?? ""))}
    if (deviceFilter.length > 0) {list = list.filter((e) => e.neteaseNumberId && deviceFilter.includes(statuses[e.neteaseNumberId]?.device_name ?? ""))}
    return list
  }, [entries, query, wbmOnly, regionFilter, deviceFilter, statuses])

  const onlineCount = useMemo(
    () => beforeOnlineFilter.filter((e) => e.neteaseNumberId && statuses[e.neteaseNumberId]?.is_online).length,
    [beforeOnlineFilter, statuses],
  )

  // Region/platform repartition among whatever's currently listed (same
  // pre-online-only-filter base as onlineCount, for the same reason —
  // shouldn't collapse once that toggle is on).
  const regionCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const e of beforeOnlineFilter) {
      const tag = e.neteaseNumberId && statuses[e.neteaseNumberId]?.oversea_tag
      if (tag) {counts[tag] = (counts[tag] ?? 0) + 1}
    }
    return Object.entries(counts).sort((a, b) => b[1] - a[1])
  }, [beforeOnlineFilter, statuses])

  const deviceCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const e of beforeOnlineFilter) {
      const device = e.neteaseNumberId && statuses[e.neteaseNumberId]?.device_name
      if (device) {counts[device] = (counts[device] ?? 0) + 1}
    }
    return Object.entries(counts).sort((a, b) => b[1] - a[1])
  }, [beforeOnlineFilter, statuses])

  const filtered = useMemo(() => {
    const list = onlineOnly ? beforeOnlineFilter.filter((e) => e.neteaseNumberId && statuses[e.neteaseNumberId]?.is_online) : beforeOnlineFilter
    return [...list].sort((a, b) => {
      if (sort === "name") {return a.name.localeCompare(b.name, undefined, { sensitivity: "base" })}
      const diff = countFor(b, sort, statuses) - countFor(a, sort, statuses)
      return diff !== 0 ? diff : a.name.localeCompare(b.name, undefined, { sensitivity: "base" })
    })
  }, [beforeOnlineFilter, onlineOnly, sort, statuses])

  const selected = useMemo(
    () => (selectedSlug ? (filtered.find((e) => e.slug === selectedSlug) ?? entries.find((e) => e.slug === selectedSlug)) : undefined),
    [selectedSlug, filtered, entries],
  )

  const isDesktop = useIsDesktop()

  useEffect(() => {
    if (!hasMountedRef.current) {
      hasMountedRef.current = true
      return
    }
    // Only relevant on desktop — mobile shows the Sheet overlay instead,
    // which needs no scroll-into-view of its own.
    if (isDesktop && selectedSlug) {detailRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" })}
  }, [selectedSlug, isDesktop])

  return (
    <div className="space-y-3">
      {/* Primary: what to browse — view toggle + search, the two controls
          almost everyone touches first. Own row, full width, so the
          search box isn't squeezed by whatever else is in the toolbar. */}
      <div className="flex flex-col sm:flex-row gap-3">
        <Tabs value={wbmOnly ? "wbm" : "all"} onValueChange={(v) => setWbmOnly(v === "wbm")}>
          <TabsList>
            <TabsTrigger value="wbm" className="inline-flex items-center gap-1.5">
              <img src={url("/images/logo_1.webp")} alt="" aria-hidden="true" className="size-5 object-contain" />
              WBM Builders
            </TabsTrigger>
            <TabsTrigger value="all" className="inline-flex items-center gap-1.5">
              <GlobeIcon weight="bold" className="size-4" />
              All Referenced Builders
            </TabsTrigger>
          </TabsList>
        </Tabs>
        <div className="relative flex-1 sm:min-w-56">
          <MagnifyingGlassIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search by Discord name, in-game name, alias, or Character ID…"
            className="pl-9"
          />
        </div>
      </div>

      {/* Secondary: how to narrow/order the results — sort, region/
          platform filters, online toggle. Grouped in their own row so
          they read as one "refine" cluster instead of competing with
          search/view-toggle for the same line. */}
      <div className="flex flex-wrap items-center gap-x-2 gap-y-2">
        <span className="text-xs font-medium text-muted-foreground shrink-0">Sort</span>
        <Tabs value={sort} onValueChange={(v) => setSort(v as SortKey)}>
          <TabsList className="flex-wrap h-auto">
            {SORT_OPTIONS.map((opt) => (
              <TabsTrigger key={opt.value} value={opt.value}>{opt.label}</TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        <span className="hidden sm:block w-px h-5 bg-border mx-1" />
        <MultiSelectFilter label="Region" options={regionOptions} selected={regionFilter} onChange={setRegionFilter} />
        <MultiSelectFilter label="Platform" options={deviceOptions} selected={deviceFilter} onChange={setDeviceFilter} formatLabel={deviceLabel} />
        <Button
          variant={onlineOnly ? "default" : "outline"}
          size="sm"
          onClick={() => setOnlineOnly((v) => !v)}
          className="gap-1.5"
        >
          <span className={`size-2 rounded-full ${onlineOnly ? "bg-white" : "bg-emerald-500"}`} />
          Online only
        </Button>
      </div>

      {/* One wrapping row instead of stacked blocks — count/online, region
          pills, and device pills only drop to their own line if the
          viewport is too narrow to hold them all, instead of always
          paying for three separate lines of vertical space. */}
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1.5 text-xs text-muted-foreground">
        <span>
          <span className="font-semibold text-foreground">{filtered.length}</span> builder{filtered.length !== 1 ? "s" : ""}
          {query && ` matching "${query}"`}
        </span>
        {beforeOnlineFilter.length > 0 && (
          <span className="inline-flex items-center gap-1">
            · <span className="size-1.5 rounded-full bg-emerald-500" /> {onlineCount}/{beforeOnlineFilter.length} online
          </span>
        )}
        {(regionCounts.length > 0 || deviceCounts.length > 0) && (
          <span className="hidden sm:block w-px h-4 bg-border mx-0.5" />
        )}
        {regionCounts.map(([region, count]) => (
          <Tag key={region} className="bg-sky-500/10 text-sky-600 dark:text-sky-300">
            {region} <span className="opacity-60">{count}</span>
          </Tag>
        ))}
        {deviceCounts.map(([device, count]) => (
          <Tag key={device} className="bg-slate-500/10 text-slate-600 dark:text-slate-300">
            {deviceLabel(device)} <span className="opacity-60">{count}</span>
          </Tag>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-5 gap-4 items-start">
        {filtered.length === 0 ? (
          <p className="text-sm text-muted-foreground py-8 text-center lg:col-span-5">No builders match your search.</p>
        ) : (
          <div className="lg:col-span-2 rounded-xl ring-1 ring-border divide-y divide-border overflow-hidden">
            {filtered.map((entry) => (
              <button
                key={entry.slug}
                type="button"
                onClick={() => setSelectedSlug(entry.slug)}
                className={`w-full flex flex-wrap items-center gap-3 px-4 py-3 text-left transition-colors cursor-pointer ${entry.slug === selectedSlug ? "bg-primary/10" : "hover:bg-muted/50"}`}
              >
                <AvatarStatus
                  src={entry.neteaseNumberId ? avatars[entry.neteaseNumberId] : undefined}
                  className="flex size-14 shrink-0"
                  levelClassName="text-[8px] px-0.5"
                  level={entry.neteaseNumberId ? statuses[entry.neteaseNumberId]?.level : undefined}
                  isOnline={entry.neteaseNumberId ? statuses[entry.neteaseNumberId]?.is_online : undefined}
                />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium truncate">{entry.name}</span>
                    {entry.isWbmBuilder && (
                      <img src={url("/images/logo_1.webp")} alt="WBM Builder" title="WBM Builder" className="size-5 object-contain shrink-0" />
                    )}
                  </div>
                  {(entry.discordName || entry.ingameNickname) && (
                    <p className="hidden sm:block text-xs text-muted-foreground truncate">
                      {[
                        entry.discordName && `Discord: ${entry.discordName}`,
                        entry.ingameNickname && entry.ingameNickname !== entry.discordName && `In-game: ${entry.ingameNickname}`,
                      ].filter(Boolean).join(" · ")}
                    </p>
                  )}
                  {/* Status tags and contribution badges both reappear
                      in full in the tap-to-open detail (Sheet on
                      mobile) — cramming them into this row too, next to
                      an already-tight avatar+name column, is what made
                      the list hard to read on narrow screens. Kept for
                      desktop, where there's room and no extra tap to see
                      them. */}
                  <div className="hidden sm:flex flex-wrap items-center gap-1 mt-1">
                    {statusTags(entry.neteaseNumberId ? statuses[entry.neteaseNumberId] : undefined)}
                  </div>
                </div>
                <div className="hidden sm:flex flex-wrap items-center gap-1.5 shrink-0">{contributionBadges(entry)}</div>
              </button>
            ))}
          </div>
        )}

        {isDesktop && (
          // top offset clears the sticky site header (h-20, shrinks to
          // h-14 on scroll) — top-4 used to tuck the panel's top edge
          // right under it, so selecting a builder left the avatar/name
          // hidden until you scrolled further. No max-h/overflow here on
          // purpose — a builder with a lot of content just scrolls the
          // page like the list column does, instead of trapping its own
          // scrollbar inside a fixed-height box.
          <div ref={detailRef} className="lg:col-span-3 rounded-xl ring-1 ring-border bg-card p-5 lg:sticky lg:top-24">
            {selected ? (
              <BuilderDetailPanel
                key={selected.slug}
                entry={selected}
                avatarUrl={selected.neteaseNumberId ? avatars[selected.neteaseNumberId] : undefined}
                status={selected.neteaseNumberId ? statuses[selected.neteaseNumberId] : undefined}
              />
            ) : (
              <p className="text-sm text-muted-foreground text-center py-8">Select a builder from the list to see their details.</p>
            )}
          </div>
        )}
      </div>

      {!isDesktop && (
        <Sheet open={!!selected} onOpenChange={(open) => !open && setSelectedSlug(null)}>
          {/* flex-col + max-h here (not overflow-y-auto directly on
              SheetContent) so only the inner div scrolls — the close
              button below is a direct SheetContent child, positioned
              absolute relative to the fixed sheet itself, so it must
              stay outside the scrolling area or it scrolls away with
              long detail content (bio/stats/links), leaving no visible
              way to dismiss once scrolled down. */}
          <SheetContent side="bottom" className="flex max-h-[85vh] flex-col rounded-t-2xl p-0">
            <SheetHeader className="sr-only">
              <SheetTitle>{selected?.name ?? "Builder details"}</SheetTitle>
            </SheetHeader>
            {selected && (
              <div className="overflow-y-auto px-4 pt-8 pb-4">
                <BuilderDetailPanel
                  key={selected.slug}
                  entry={selected}
                  avatarUrl={selected.neteaseNumberId ? avatars[selected.neteaseNumberId] : undefined}
                  status={selected.neteaseNumberId ? statuses[selected.neteaseNumberId] : undefined}
                />
              </div>
            )}
          </SheetContent>
        </Sheet>
      )}
    </div>
  )
}
