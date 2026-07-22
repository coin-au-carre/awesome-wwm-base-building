import * as React from "react"
import { useEffect, useRef, useState } from "react"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from "@/components/ui/collapsible"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { HammerIcon, HeartIcon, FireIcon, DownloadSimpleIcon, CaretLeftIcon, CaretRightIcon, CheckIcon, CopyIcon, ShareNetworkIcon, UserCircleIcon, LockIcon, GlobeIcon, UsersIcon, MagnifyingGlassIcon, FactoryIcon, TrendUpIcon, CalendarIcon, ChatCircleIcon, InfoIcon } from "@phosphor-icons/react"
import { buttonVariants } from "@/components/ui/button"
import { url } from "@/lib/url"
import { relativeTime } from "@/lib/dates"
import {
  WBM_RELAY_URL,
  SORT_OPTIONS,
  WBM_SORT_OPTIONS,
  DEFAULT_SORT,
  CATEGORY_OPTIONS,
  searchGalleryUrl,
  wbmGalleryUrl,
  isValidGalleryCode,
  designerUrl,
  commentsUrl,
  formatCount,
  categoryLabel,
  isPrivate,
  planVisibility,
  type GalleryPlan,
  type PlanDetail,
  type DesignerProfile,
  type Comment,
} from "@/lib/gallery"

const VISIBILITY_STYLE = {
  private: { icon: LockIcon, label: "Private", className: "bg-amber-500/80 text-white", title: "Only visible to the designer" },
  friends: { icon: UsersIcon, label: "Friends Can Apply", className: "bg-sky-500/80 text-white", title: "Friends Can Apply — best-effort detection, not fully confirmed yet" },
  public: { icon: GlobeIcon, label: "Public/Friends Only/Cannot Apply", className: "bg-black/60 text-white/90", title: "Indistinguishable in the data: Public, Cannot Apply, and (since Friends Can Apply detection is best-effort) possibly Friends Can Apply too" },
} as const

// Visibility pill. Grid thumbnails only ever know the coarse `private`
// flag (hasFriendsWhitelist omitted); the plan detail view has the
// fuller PlanDetail and can tell "Friends Only" apart too — see
// lib/gallery.ts's planVisibility.
export function VisibilityBadge({
  private_,
  hasFriendsWhitelist = false,
  size = "sm",
  className = "",
}: {
  private_: number
  hasFriendsWhitelist?: boolean
  size?: "sm" | "md"
  className?: string
}) {
  const visibility = planVisibility(private_, hasFriendsWhitelist)
  const state = VISIBILITY_STYLE[visibility]
  const Icon = state.icon
  const sizeClasses = size === "md" ? "text-sm px-2 py-1 gap-1.5" : "text-[11px] px-2 py-0.5 gap-1"
  const iconSize = size === "md" ? "size-4" : "size-3"
  // The generic "public" catch-all is the common case and not worth the
  // screen space on small viewports — private/friends-can-apply are
  // actually useful info, so those stay visible everywhere.
  const displayClasses = visibility === "public" ? "hidden sm:inline-flex" : "inline-flex"
  return (
    <span
      title={state.title}
      className={`${displayClasses} items-center font-medium rounded-md backdrop-blur-sm ${sizeClasses} ${state.className} ${className}`}
    >
      <Icon weight="fill" className={iconSize} />
      {state.label}
    </span>
  )
}

const LIMIT = 20

export function StatRow({
  plan,
  className = "",
  compact = false,
}: {
  plan: Pick<GalleryPlan, "heat_val" | "like_num" | "build_num" | "components_count">
  className?: string
  // compact hides the like/download counts on mobile (grid cards, tight
  // on space) — the diagram detail view has room to always show them.
  compact?: boolean
}) {
  // Likes are the least essential stat, so they get the widest hidden
  // range (mobile + tablet, only shown at lg+); downloads only hide on
  // mobile — both stay always-visible outside compact (grid card) use.
  const hideLikes = compact ? "hidden lg:flex" : "flex"
  const hideDownloads = compact ? "hidden sm:flex" : "flex"
  return (
    <div className={`flex flex-wrap items-center gap-x-3 gap-y-0.5 ${className}`}>
      <span className="flex items-center gap-1">
        <FireIcon weight="fill" className="size-3.5 text-orange-400" /> {formatCount(plan.heat_val)}
      </span>
      <span className={`${hideLikes} items-center gap-1`}>
        <HeartIcon weight="fill" className="size-3.5 text-rose-400" /> {formatCount(plan.like_num)}
      </span>
      <span className={`${hideDownloads} items-center gap-1`} title="Downloads">
        <DownloadSimpleIcon weight="bold" className="size-3.5" /> {formatCount(plan.build_num)}
      </span>
      {plan.components_count != null && plan.components_count > 0 && (
        <span className="flex items-center gap-1 font-semibold" title="Components used">
          <HammerIcon weight="duotone" className="size-3.5 text-violet-400" /> {formatCount(plan.components_count)}
        </span>
      )}
    </div>
  )
}

// Every image for the detail view's carousel: the cover picture first,
// then any additional previews (deduped — the upstream API sometimes
// repeats the cover inside `previews`).
function detailImages(detail: PlanDetail): string[] {
  const urls = [detail.picture_url, ...(detail.previews ?? [])].filter(Boolean)
  return Array.from(new Set(urls))
}

// A builder's real in-game character portrait when resolved (see
// wbm-relay's avatar.go), falling back to the generic icon otherwise —
// used anywhere a specific author/designer is shown (comment rows, plan
// detail author link, builder profile header).
export function Avatar({ src, className = "size-8" }: { src?: string; className?: string }) {
  return src ? (
    <img
      src={src}
      alt=""
      className={`${className} rounded-full object-cover bg-muted shrink-0`}
      loading="lazy"
      onError={(e) => (e.currentTarget.style.display = "none")}
    />
  ) : (
    <UserCircleIcon weight="fill" className={`${className} text-muted-foreground/50 shrink-0`} />
  )
}

// One comment row (avatar + builder link + relative time + message) —
// shared between the always-visible first 10 and the collapsible rest,
// see PlanDetailContent's comments section.
function CommentRow({ comment: c }: { comment: Comment }) {
  return (
    <div className="flex items-start gap-2.5">
      <a href={builderHref(c.author_number_id)} className="shrink-0">
        <Avatar src={c.author_avatar_url} className="size-8" />
      </a>
      <div className="min-w-0">
        <div className="flex items-baseline gap-2">
          <a href={builderHref(c.author_number_id)} className="text-sm font-medium hover:text-primary transition-colors truncate">
            {c.author_name || c.author_number_id || "Unknown builder"}
          </a>
          <span className="text-xs text-muted-foreground shrink-0">{relativeTime(c.ts * 1000)}</span>
        </div>
        <p className="text-sm text-foreground/90 whitespace-pre-line wrap-break-word">{c.msg}</p>
      </div>
    </div>
  )
}

// A small stat card for the plan detail meta grid (Components/Industry
// level/Prosperity/Downloads/Created) — icon + label + value, replacing
// what used to be a plain flex row of "Label value" text pairs.
function StatTile({
  icon: Icon,
  label,
  value,
}: {
  icon: React.ComponentType<{ weight?: "regular" | "bold" | "duotone" | "fill"; className?: string }>
  label: string
  value: string | number
}) {
  return (
    <div className="flex items-center gap-2.5 rounded-lg border border-border bg-muted/30 px-3 py-2">
      <Icon weight="duotone" className="size-5 text-muted-foreground shrink-0" />
      <div className="min-w-0">
        <div className="text-[11px] uppercase tracking-wide text-muted-foreground leading-none">{label}</div>
        <div className="text-sm font-semibold leading-tight whitespace-nowrap">{value}</div>
      </div>
    </div>
  )
}

// A code pill (art_code / share_id) that copies its own value on click.
export function CopyPill({ label, value, className = "" }: { label: string; value: string; className?: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <button
      onClick={(e) => {
        e.stopPropagation()
        navigator.clipboard.writeText(value)
        setCopied(true)
        setTimeout(() => setCopied(false), 800)
      }}
      title={`Copy ${label}`}
      className={`inline-flex items-center gap-2 rounded-full border border-border bg-muted/50 px-4 py-1.5 text-sm font-mono cursor-pointer select-none hover:bg-muted transition-colors ${className}`}
    >
      <span className="text-[10px] uppercase tracking-wider text-muted-foreground">{label}</span>
      {value}
      {copied ? (
        <CheckIcon weight="bold" className="size-3.5 text-green-500" />
      ) : (
        <CopyIcon weight="bold" className="size-3.5 text-muted-foreground" />
      )}
    </button>
  )
}

// Copies the current page's own URL — used on the shareable plan/
// builder/gallery pages so visitors have an obvious "share this" action
// rather than having to copy the address bar themselves. location.href
// is only read at click/render time, both of which happen well after
// hydration on these pages (they only render this once their data has
// loaded), so it's never touched during Astro's server/build pass.
// showLink additionally always displays a readonly input with the exact
// link, instead of just a toast on click, so it's obvious up front that
// a share link is a snapshot of the current view (filters/sort/search),
// not just the bare page URL — used on the gallery grid. When showLink
// is set, pass href explicitly and keep it in sync with whatever state
// drives the URL (e.g. GalleryGrid's own sort/tag/wbmOnly) — reading
// location.href here instead would go stale after a filter change
// without a full re-render/navigation to refresh it.
export function ShareButton({
  label = "Share",
  variant = "outline",
  showLink = false,
  href,
}: {
  label?: string
  variant?: "outline" | "default"
  showLink?: boolean
  href?: string
}) {
  const [copied, setCopied] = useState(false)

  function copy(value: string) {
    navigator.clipboard.writeText(value)
    setCopied(true)
    setTimeout(() => setCopied(false), 1200)
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      {!showLink && (
        <button onClick={() => copy(location.href)} className={buttonVariants({ variant, size: "sm" })}>
          {copied ? (
            <>
              <CheckIcon weight="bold" className="size-3.5 text-green-500" /> Copied!
            </>
          ) : (
            <>
              <ShareNetworkIcon weight="bold" className="size-3.5" /> {label}
            </>
          )}
        </button>
      )}
      {showLink && href !== undefined && (
        <div className="flex items-center gap-2 w-full sm:w-auto">
          {/* Plain <input>, not the shadcn Input wrapper — that component
              doesn't declare a ref param, so a scroll-to-end callback ref
              wouldn't actually reach the DOM node. Class list mirrors
              Input's own recipe. */}
          <input
            readOnly
            value={href}
            // The host/path prefix never changes — what's actually worth
            // seeing is the query string (filters/sort/search) at the
            // end, so scroll there by default instead of showing the
            // same-looking start of every link.
            ref={(el) => {
              if (el) el.scrollLeft = el.scrollWidth
            }}
            onFocus={(e) => e.currentTarget.select()}
            className="h-8 sm:w-80 w-full min-w-0 rounded-md border border-input bg-transparent px-3 font-mono text-xs shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
          />
          <button onClick={() => copy(href)} className={buttonVariants({ variant, size: "sm", className: "shrink-0" })}>
            {copied ? (
              <>
                <CheckIcon weight="bold" className="size-3.5 text-green-500" /> Copied!
              </>
            ) : (
              <>
                <ShareNetworkIcon weight="bold" className="size-3.5" /> {label}
              </>
            )}
          </button>
        </div>
      )}
    </div>
  )
}

// A builder link — same shape whether it comes off a gallery card
// (GalleryPlan) or a designer's own build grid (DesignerPlan). numberID
// is the public account number — wbm-relay resolves whatever internal
// id it needs server-side.
export function builderHref(numberID?: string): string {
  return url(`/gallery/builder?id=${encodeURIComponent(numberID ?? "")}`)
}

// A query is only ever treated as an ART/SHARE code attempt if it starts
// like one — anything else (a builder's name, a bare numeric id) is left
// alone for the Enter-to-navigate path below instead of being live-
// validated/searched as a code. Prefix-only (not full isValidGalleryCode)
// so "not a valid code" feedback only fires once someone's clearly
// typing a code and getting it wrong, not while typing an unrelated name.
function looksLikeCodeAttempt(v: string): boolean {
  const lower = v.toLowerCase()
  return lower.startsWith("art") || lower.startsWith("share")
}

// A shareable link to one diagram's own page (gallery/diagram.astro).
// shareId is the SHARE code (not plan_id/art_code) — the same code the
// in-game "share" button produces, more recognizable to players than
// an internal id. wbm-relay's GET /api/plan resolves it server-side.
export function planHref(shareId?: string): string {
  return url(`/gallery/diagram?share=${encodeURIComponent(shareId ?? "")}`)
}

// One gallery thumbnail card — used by the main grid, a builder's own
// build grid, and "more by this builder" on the plan detail page.
// showAuthor is off on a builder's own grid (redundant — already
// viewing that builder) and on "more by this builder" (same reason).
// wbmSlug, when present, means this plan's author is also a known WBM
// builder (see data/builder_identities.json) — see docs/builder-identity.md.
// onWbmBadgeClick, when given, turns the WBM badge into a filter toggle
// (used on the gallery grid itself) instead of its default behavior of
// linking out to the builder's full WBM profile (used everywhere else
// PlanCard shows up — "more by this builder", the builder profile's own
// gallery preview — where there's no WBM-only filter to toggle).
export function PlanCard({
  plan,
  showAuthor = true,
  wbmSlug,
  onWbmBadgeClick,
}: {
  plan: GalleryPlan
  showAuthor?: boolean
  wbmSlug?: string
  onWbmBadgeClick?: () => void
}) {
  const label = categoryLabel(plan.category_tag)
  return (
    <div className="group relative overflow-hidden rounded-xl ring-1 ring-border aspect-video bg-muted">
      {/* The card's own link — wraps everything except the author badge
          and the WBM logo badge below, which must stay siblings (not
          nested) since they're their own separate links. */}
      <a href={planHref(plan.share_id)} aria-label={plan.art_code} className="absolute inset-0">
        {plan.picture_url && (
          <img
            src={plan.picture_url}
            alt=""
            loading="lazy"
            className="absolute inset-0 h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
            onError={(e) => (e.currentTarget.style.display = "none")}
          />
        )}
        <div className="absolute inset-0 bg-linear-to-t from-black/70 to-transparent" />
        {label && (
          <span className="absolute top-2 left-2 text-[11px] font-medium px-2 py-0.5 rounded-md bg-black/60 text-white/90 backdrop-blur-sm">
            {label}
          </span>
        )}
        {!wbmSlug && isPrivate(plan.private) && (
          <div className="absolute top-2 right-2">
            <VisibilityBadge private_={plan.private} />
          </div>
        )}
        <div className="absolute bottom-0 left-0 p-3">
          <StatRow plan={plan} className="text-xs text-white/90" compact />
        </div>
      </a>
      {wbmSlug && (onWbmBadgeClick ? (
        <button
          type="button"
          onClick={onWbmBadgeClick}
          title="Show only WBM Builders"
          className="absolute top-2 right-2 z-10 flex items-center gap-1 cursor-pointer"
        >
          <img src={url("/images/logo_1.webp")} alt="WBM Builder" className="size-9 rounded-full bg-black/60 backdrop-blur-sm ring-1 ring-white/20 object-contain p-1 hover:ring-primary transition-all" />
          {isPrivate(plan.private) && <VisibilityBadge private_={plan.private} />}
        </button>
      ) : (
        <a
          href={url(`/builders/${wbmSlug}`)}
          title="WBM Builder"
          className="absolute top-2 right-2 z-10 flex items-center gap-1"
        >
          <img src={url("/images/logo_1.webp")} alt="WBM Builder" className="size-9 rounded-full bg-black/60 backdrop-blur-sm ring-1 ring-white/20 object-contain p-1 hover:ring-primary transition-all" />
          {isPrivate(plan.private) && <VisibilityBadge private_={plan.private} />}
        </a>
      ))}
      {showAuthor && plan.author_name && (
        <a
          href={builderHref(plan.author_number_id)}
          className="absolute bottom-2 right-2 md:bottom-2.5 md:right-2.5 lg:bottom-3 lg:right-3 z-10 inline-flex items-center gap-1.5 md:gap-2 text-xs md:text-sm lg:text-base font-semibold text-white bg-black/50 hover:bg-primary hover:text-primary-foreground backdrop-blur-sm rounded-full pl-1 pr-2 lg:pr-3 py-0.5 lg:py-1 truncate max-w-28 md:max-w-32 lg:max-w-40 shrink-0 transition-colors"
        >
          {plan.author_avatar_url && <Avatar src={plan.author_avatar_url} className="size-5 md:size-6 lg:size-8" />}
          {plan.author_name}
        </a>
      )}
    </div>
  )
}

// The full content of one plan's detail view — image carousel, author
// link, stats, component breakdown, code pills. Used both by the
// standalone shareable page (PlanPage) and could be reused anywhere
// else a full detail view is needed; deliberately has no modal/page
// chrome of its own; note it renders its own author link, so callers
// must not also nest this inside another <a>.
export function PlanDetailContent({ detail, wbmSlug }: { detail: PlanDetail; wbmSlug?: string }) {
  const [imgIndex, setImgIndex] = useState(0)
  const images = detailImages(detail)

  // Auto-advance every 4s, paused on hover so it doesn't fight anyone
  // actually looking at a specific screenshot. No-op for a single image.
  const [paused, setPaused] = useState(false)
  useEffect(() => {
    if (images.length <= 1 || paused) return
    const t = setInterval(() => setImgIndex((i) => (i + 1) % images.length), 4000)
    return () => clearInterval(t)
  }, [images.length, paused])

  // "More by this builder" — fetched separately from detail itself
  // (which only carries author identity, not their other plans).
  // null = not loaded/loading yet, [] = loaded but nothing to show
  // (own request failed or no other plans) — both render nothing, no
  // separate error state needed since this is supplementary, not core
  // content.
  const [moreByBuilder, setMoreByBuilder] = useState<GalleryPlan[] | null>(null)
  useEffect(() => {
    setMoreByBuilder(null)
    if (!detail.author_number_id) return
    fetch(designerUrl(detail.author_number_id))
      .then((res) => (res.ok ? (res.json() as Promise<DesignerProfile>) : Promise.reject()))
      .then((profile) => {
        const others = profile.plans
          .filter((p) => p.plan_id !== detail.plan_id)
          .sort((a, b) => b.upload_ts - a.upload_ts)
          .slice(0, 6)
        setMoreByBuilder(others)
      })
      .catch(() => setMoreByBuilder([]))
  }, [detail.author_number_id, detail.plan_id])

  // Same null/[] convention as moreByBuilder above — supplementary, not
  // core content, so a failed fetch just renders nothing.
  const [comments, setComments] = useState<Comment[] | null>(null)
  useEffect(() => {
    setComments(null)
    if (!detail.plan_id) return
    fetch(commentsUrl(detail.plan_id))
      .then((res) => (res.ok ? (res.json() as Promise<Comment[]>) : Promise.reject()))
      .then(setComments)
      .catch(() => setComments([]))
  }, [detail.plan_id])

  return (
    <>
      <div className="relative" onMouseEnter={() => setPaused(true)} onMouseLeave={() => setPaused(false)}>
        {images[imgIndex] && (
          <img
            src={images[imgIndex]}
            alt={detail.title}
            className="w-full max-h-[65vh] object-contain bg-black"
          />
        )}
        {images.length > 1 && (
          <>
            <button
              onClick={() => setImgIndex((i) => (i - 1 + images.length) % images.length)}
              aria-label="Previous image"
              className="absolute left-2 top-1/2 -translate-y-1/2 flex items-center justify-center size-8 rounded-full bg-black/50 hover:bg-black/70 text-white transition-colors cursor-pointer"
            >
              <CaretLeftIcon weight="bold" className="size-4" />
            </button>
            <button
              onClick={() => setImgIndex((i) => (i + 1) % images.length)}
              aria-label="Next image"
              className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center justify-center size-8 rounded-full bg-black/50 hover:bg-black/70 text-white transition-colors cursor-pointer"
            >
              <CaretRightIcon weight="bold" className="size-4" />
            </button>
            <div className="absolute top-3 left-1/2 -translate-x-1/2 flex gap-1.5">
              {images.map((_, i) => (
                <button
                  key={i}
                  onClick={() => setImgIndex(i)}
                  aria-label={`Image ${i + 1}`}
                  className={`size-1.5 rounded-full transition-colors cursor-pointer ${i === imgIndex ? "bg-white" : "bg-white/40 hover:bg-white/60"}`}
                />
              ))}
            </div>
          </>
        )}
        <div className="absolute inset-x-0 bottom-0 bg-linear-to-t from-black/85 via-black/40 to-transparent px-6 pt-10 pb-5">
          {detail.title && (
            <h2 className="font-heading text-xl sm:text-2xl font-bold text-white drop-shadow leading-tight">
              {detail.title}
            </h2>
          )}
          <StatRow plan={detail} className="mt-2 text-sm text-white/85" />
        </div>
      </div>
      <div className="p-6 space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center gap-3">
          {detail.author_name && (
            <a
              href={builderHref(detail.author_number_id)}
              className={buttonVariants({ variant: "secondary", size: "lg", className: "h-auto py-1.5 text-base sm:text-lg font-semibold justify-start sm:justify-center" })}
            >
              <Avatar src={detail.author_avatar_url} className="size-9" />
              {detail.author_name}
              <CaretRightIcon weight="bold" className="size-4 opacity-60" />
            </a>
          )}
          {wbmSlug && (
            <a href={url(`/builders/${wbmSlug}`)} title="WBM Builder — see full profile">
              <img src={url("/images/logo_1.webp")} alt="WBM Builder" className="size-9 rounded-full bg-muted ring-1 ring-border object-contain p-1 hover:ring-primary transition-all" />
            </a>
          )}
          <div className="flex items-center gap-2 flex-wrap sm:ml-auto">
            {detail.author_number_id && <CopyPill label="ID" value={detail.author_number_id} />}
            <div className="hidden sm:block">
              <CopyPill label="ART" value={detail.art_code} />
            </div>
            {detail.share_id && <CopyPill label="Share" value={detail.share_id} />}
          </div>
        </div>
        <div className="flex items-start justify-between gap-3">
          {detail.description && (
            <p className="text-sm text-muted-foreground leading-relaxed whitespace-pre-line">
              {detail.description}
            </p>
          )}
          <VisibilityBadge
            private_={detail.private}
            hasFriendsWhitelist={detail.has_friends_whitelist}
            size="md"
            className="ml-auto shrink-0 bg-muted/50! text-foreground! border border-border"
          />
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {detail.components_count != null && detail.components_count > 0 && (
            <StatTile icon={HammerIcon} label="Components" value={detail.components_count} />
          )}
          {detail.industry_level !== undefined && (
            <StatTile icon={FactoryIcon} label="Industry level" value={detail.industry_level} />
          )}
          {detail.prosperity !== undefined && (
            <StatTile icon={TrendUpIcon} label="Prosperity" value={detail.prosperity} />
          )}
          {detail.upload_ts > 0 && (
            <StatTile icon={CalendarIcon} label="Created" value={new Date(detail.upload_ts * 1000).toLocaleDateString()} />
          )}
        </div>
        {detail.components && Object.keys(detail.components).length > 0 && (
          <Collapsible>
            <CollapsibleTrigger className="group flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
              <CaretRightIcon weight="bold" className="size-3 transition-transform group-data-[state=open]:rotate-90" />
              Component list ({Object.keys(detail.components).length} types)
            </CollapsibleTrigger>
            <CollapsibleContent>
              <p className="text-xs text-muted-foreground mt-2 mb-1">
                Raw internal item codes — no name mapping exists yet, so these aren't human-readable item names.
              </p>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-4 gap-y-1 text-sm font-mono">
                {Object.entries(detail.components)
                  .sort((a, b) => b[1] - a[1])
                  .map(([code, count]) => (
                    <div key={code} className="flex justify-between">
                      <span className="text-muted-foreground">#{code}</span>
                      <span>×{count}</span>
                    </div>
                  ))}
              </div>
            </CollapsibleContent>
          </Collapsible>
        )}
      </div>
      {((comments && comments.length > 0) || (moreByBuilder && moreByBuilder.length > 0)) && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 p-6 pt-0">
          {comments && comments.length > 0 && (
            <div className="space-y-3">
              <h3 className="flex items-center gap-1.5 text-sm font-semibold text-muted-foreground">
                <ChatCircleIcon weight="fill" className="size-4" />
                Comments ({comments.length})
              </h3>
              <div className="space-y-3">
                {comments.slice(0, 10).map((c) => (
                  <CommentRow key={c.comment_id} comment={c} />
                ))}
              </div>
              {comments.length > 10 && (
                <Collapsible>
                  <CollapsibleTrigger className="group flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
                    <CaretRightIcon weight="bold" className="size-3 transition-transform group-data-[state=open]:rotate-90" />
                    Show {comments.length - 10} more comment{comments.length - 10 === 1 ? "" : "s"}
                  </CollapsibleTrigger>
                  <CollapsibleContent className="space-y-3 mt-3">
                    {comments.slice(10).map((c) => (
                      <CommentRow key={c.comment_id} comment={c} />
                    ))}
                  </CollapsibleContent>
                </Collapsible>
              )}
            </div>
          )}
          {moreByBuilder && moreByBuilder.length > 0 && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-muted-foreground">
                  More by {detail.author_name || "this builder"}
                </h3>
                <a href={builderHref(detail.author_number_id)} className="text-xs text-muted-foreground hover:text-foreground transition-colors">
                  View all
                </a>
              </div>
              <div className="grid grid-cols-2 gap-3">
                {moreByBuilder.map((p) => (
                  <PlanCard key={p.plan_id} plan={p} showAuthor={false} />
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </>
  )
}

// Reads a param from the current URL at mount time only (SSR-safe:
// window is guarded), falling back when absent/invalid — used to seed
// sort/tag state so filters survive a refresh or a shared link.
function initFromQuery<T>(key: string, valid: readonly T[], fallback: T): T {
  if (typeof window === "undefined") return fallback
  const raw = new URLSearchParams(window.location.search).get(key)
  const match = valid.find((v) => String(v) === raw)
  return match ?? fallback
}

// Same idea as initFromQuery, but for a free-form non-negative integer
// (min/max components) rather than a fixed set of valid values.
function initIntFromQuery(key: string): number | undefined {
  if (typeof window === "undefined") return undefined
  const raw = new URLSearchParams(window.location.search).get(key)
  const n = raw ? Number(raw) : NaN
  return Number.isInteger(n) && n >= 0 ? n : undefined
}

// Builds the ?sort&tag&view&min&max query string for the current filter
// state, omitting params at their default to keep plain /gallery links
// clean — shared by the URL-sync effect and the Share Gallery link so
// they can't drift out of sync with each other.
function galleryFilterQuery(
  sort: string,
  tag: number,
  wbmOnly: boolean,
  minComponents?: number,
  maxComponents?: number,
): string {
  const params = new URLSearchParams()
  if (sort !== DEFAULT_SORT) params.set("sort", sort)
  if (tag !== CATEGORY_OPTIONS[0].value) params.set("tag", String(tag))
  if (!wbmOnly) params.set("view", "all")
  // min/max only ever apply to the WBM-only view (see wbmGalleryUrl) —
  // omit them from the URL when All Builders is active so a shared link
  // doesn't carry a filter that silently does nothing.
  if (wbmOnly && minComponents != null) params.set("min", String(minComponents))
  if (wbmOnly && maxComponents != null) params.set("max", String(maxComponents))
  return params.toString()
}

// wbmBuilders maps a NetEase author_number_id to their WBM canonicalSlug
// — see data/builder_identities.json, loaded server-side in gallery.astro.
export function GalleryGrid({ wbmBuilders = {} }: { wbmBuilders?: Record<string, string> }) {
  const [wbmOnly, setWbmOnly] = useState(() => initFromQuery("view", ["wbm", "all"], "wbm") === "wbm")
  const [sort, setSort] = useState<string>(() =>
    initFromQuery("sort", (wbmOnly ? WBM_SORT_OPTIONS : SORT_OPTIONS).map((o) => o.value), DEFAULT_SORT),
  )
  const [tag, setTag] = useState<number>(() =>
    initFromQuery("tag", CATEGORY_OPTIONS.map((o) => o.value), CATEGORY_OPTIONS[0].value),
  )
  // Only meaningful/sent when wbmOnly (see wbmGalleryUrl) — component
  // count range filter, WBM-only aggregate can filter this in-memory,
  // the general feed has no server-side way to.
  const [minComponents, setMinComponents] = useState<number | undefined>(() => initIntFromQuery("min"))
  const [maxComponents, setMaxComponents] = useState<number | undefined>(() => initIntFromQuery("max"))
  // Raw text, debounced into the numeric state above — same rationale as
  // the search box's debouncedQuery: don't refetch on every keystroke.
  const [minInput, setMinInput] = useState(minComponents != null ? String(minComponents) : "")
  const [maxInput, setMaxInput] = useState(maxComponents != null ? String(maxComponents) : "")
  useEffect(() => {
    const t = setTimeout(() => {
      const n = minInput.trim() === "" ? undefined : Number(minInput)
      setMinComponents(n != null && Number.isFinite(n) && n >= 0 ? Math.floor(n) : undefined)
    }, 400)
    return () => clearTimeout(t)
  }, [minInput])
  useEffect(() => {
    const t = setTimeout(() => {
      const n = maxInput.trim() === "" ? undefined : Number(maxInput)
      setMaxComponents(n != null && Number.isFinite(n) && n >= 0 ? Math.floor(n) : undefined)
    }, 400)
    return () => clearTimeout(t)
  }, [maxInput])
  const [query, setQuery] = useState("")
  const [plans, setPlans] = useState<GalleryPlan[]>([])
  const [nextStart, setNextStart] = useState(0)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [hasMore, setHasMore] = useState(true)
  const sentinelRef = useRef<HTMLDivElement | null>(null)

  // "components" only exists on the WBM aggregate endpoint (see
  // WBM_SORT_OPTIONS in lib/gallery.ts) — falls back to the default sort
  // rather than sending an invalid value to /api/gallery if the toggle
  // switches away from WBM Builders while it's selected.
  useEffect(() => {
    if (!wbmOnly && sort === "components") setSort(DEFAULT_SORT)
  }, [wbmOnly, sort])

  // Same reasoning — the components range filter only exists on the WBM
  // aggregate endpoint, clear it rather than silently keep sending it
  // (harmlessly ignored by /api/gallery, but confusing left set in the UI).
  useEffect(() => {
    if (!wbmOnly && (minInput !== "" || maxInput !== "")) {
      setMinInput("")
      setMaxInput("")
    }
  }, [wbmOnly, minInput, maxInput])

  // Debounce the raw input into a value the fetch effect reacts to, so
  // we don't hit the relay on every keystroke.
  const [debouncedQuery, setDebouncedQuery] = useState("")
  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query.trim()), 400)
    return () => clearTimeout(t)
  }, [query])

  // The search box is shared between two unrelated intents — an
  // ART/SHARE code (live-searched here, debounced) vs. a builder
  // name/id (Enter navigates to their profile instead, see
  // handleSearchKeyDown) — so only a code-shaped attempt ever engages
  // the code-search path at all. Typing an ordinary builder name leaves
  // the grid alone (still showing the current WBM/All feed) rather than
  // clearing it or flashing a "not a valid code" error.
  const codeAttempt = looksLikeCodeAttempt(debouncedQuery)
  const queryInvalid = codeAttempt && !isValidGalleryCode(debouncedQuery)
  const activeSearch = isValidGalleryCode(debouncedQuery) ? debouncedQuery : ""

  // sort/tag/search/wbmOnly change → reset and refetch page 0. An active
  // code search takes over the whole grid (single page, no pagination —
  // see wbm-relay's /api/search, only confirmed for ART/SHARE codes so
  // far) and wins over wbmOnly, same precedence as sort/tag already had.
  // wbmOnly (when a search isn't active) hits a separate, purpose-built
  // endpoint (wbmGalleryUrl) that applies the same sort/tag values to its
  // own cached aggregate of known WBM builders' plans, rather than
  // filtering the general feed client-side (which would mean fetching
  // and discarding most of each page).
  useEffect(() => {
    if (queryInvalid) {
      setLoading(false)
      setPlans([])
      setHasMore(false)
      setError(null)
      return
    }
    setLoading(true)
    setError(null)
    setHasMore(true)
    if (!WBM_RELAY_URL) {
      setLoading(false)
      setHasMore(false)
      setError("not deployed yet")
      return
    }
    const fetchUrl = activeSearch
      ? searchGalleryUrl(activeSearch)
      : wbmOnly
        ? wbmGalleryUrl(sort, tag, 0, LIMIT, minComponents, maxComponents)
        : `${WBM_RELAY_URL}/api/gallery?sort=${sort}&tag=${tag}&start=0&limit=${LIMIT}`
    fetch(fetchUrl)
      .then((res) => {
        if (!res.ok) {
          throw new Error(`relay returned ${res.status}`)
        }
        return res.json()
      })
      .then((data) => {
        const fetched = data.plans ?? []
        setPlans(fetched)
        setNextStart(data.next_start ?? 0)
        setHasMore(!activeSearch && fetched.length > 0)
      })
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false))
  }, [sort, tag, activeSearch, queryInvalid, wbmOnly, minComponents, maxComponents])

  // Keep the URL in sync so filters survive a refresh and can be
  // shared/bookmarked — omits params at their default to keep plain
  // /gallery links clean.
  const filterQuery = galleryFilterQuery(sort, tag, wbmOnly, minComponents, maxComponents)
  useEffect(() => {
    // Preserve Astro ClientRouter's own history.state (index/scroll
    // position) — replacing it with null breaks browser back/forward
    // across page transitions, since the router keys off state.index.
    history.replaceState(history.state, "", filterQuery ? `?${filterQuery}` : window.location.pathname)
  }, [filterQuery])

  // Derived straight from sort/tag/wbmOnly state (not read back off
  // location.href), so the Share Gallery input stays correct the
  // instant a filter changes rather than depending on the effect above
  // having already run this render.
  const shareUrl =
    typeof window === "undefined"
      ? undefined
      : `${window.location.origin}${window.location.pathname}${filterQuery ? `?${filterQuery}` : ""}`

  function loadMore() {
    setLoadingMore(true)
    const fetchUrl = wbmOnly
      ? wbmGalleryUrl(sort, tag, nextStart, LIMIT, minComponents, maxComponents)
      : `${WBM_RELAY_URL}/api/gallery?sort=${sort}&tag=${tag}&start=${nextStart}&limit=${LIMIT}`
    fetch(fetchUrl)
      .then((res) => {
        if (!res.ok) {
          throw new Error(`relay returned ${res.status}`)
        }
        return res.json()
      })
      .then((data) => {
        const fetched = data.plans ?? []
        setPlans((prev) => [...prev, ...fetched])
        setNextStart(data.next_start ?? nextStart)
        setHasMore(fetched.length > 0)
      })
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoadingMore(false))
  }

  // Scroll-triggered pagination — same idea as moments.astro's
  // lightbox neighbour-preloading, but for whole pages: an
  // IntersectionObserver on a sentinel below the grid calls loadMore()
  // once it's near-visible, instead of a manual button.
  useEffect(() => {
    const sentinel = sentinelRef.current
    if (!sentinel || loading || loadingMore || !hasMore) {
      return
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          loadMore()
        }
      },
      { rootMargin: "600px" },
    )
    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [loading, loadingMore, hasMore, nextStart])

  // Enter navigates straight to a builder's profile — only when the box
  // isn't holding a code attempt, which stays live-searched in place
  // instead (see codeAttempt/activeSearch above).
  function handleSearchKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key !== "Enter") return
    const v = query.trim()
    if (!v || looksLikeCodeAttempt(v)) return
    const param = /^\d+$/.test(v) ? "id" : "name"
    location.href = url(`/gallery/builder?${param}=${encodeURIComponent(v)}`)
  }

  return (
    <div className="space-y-4">
      {/* Primary choice: which gallery this is — not just another filter,
          so it gets top billing over search/category/sort below. */}
      <div className="flex flex-wrap items-center justify-between gap-2">
        <Tabs value={wbmOnly ? "wbm" : "all"} onValueChange={(v) => setWbmOnly(v === "wbm")}>
          <TabsList>
            <TabsTrigger value="wbm" className="inline-flex items-center gap-1.5">
              <img src={url("/images/logo_1.webp")} alt="" aria-hidden="true" className="size-4 object-contain" />
              WBM Builders
              <span className="text-[10px] opacity-60 font-normal">· {Object.keys(wbmBuilders).length}</span>
            </TabsTrigger>
            <TabsTrigger value="all" className="inline-flex items-center gap-1.5">
              <GlobeIcon weight="bold" className="size-4" />
              Global Builders
            </TabsTrigger>
          </TabsList>
        </Tabs>
        <ShareButton label="Share Gallery" variant="default" showLink href={shareUrl} />
      </div>

      <div>
        <div className="relative">
          <MagnifyingGlassIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleSearchKeyDown}
            placeholder="Search by builder name/ID, ART or SHARE code…"
            className="pl-9"
            aria-invalid={queryInvalid}
          />
        </div>
        {queryInvalid && (
          <p className="mt-1 text-xs text-amber-600 dark:text-amber-400">
            Not a valid ART or SHARE code (e.g. ARTakLUQfFVevW1Xl1A or SHARE5f223181ad510813).
          </p>
        )}
      </div>

      {!query && (
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Select value={String(tag)} onValueChange={(v) => setTag(Number(v))}>
            <SelectTrigger size="sm" className="sm:hidden">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {CATEGORY_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={String(opt.value)}>{opt.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <div className="hidden sm:flex flex-wrap gap-1.5">
            {CATEGORY_OPTIONS.map((opt) => (
              <Badge key={opt.value} variant={tag === opt.value ? "default" : "outline"} asChild>
                <button onClick={() => setTag(opt.value)} className="cursor-pointer">
                  {opt.label}
                </button>
              </Badge>
            ))}
          </div>
          <Tabs value={sort} onValueChange={setSort}>
            <TabsList>
              {(wbmOnly ? WBM_SORT_OPTIONS : SORT_OPTIONS).map((opt) => (
                <TabsTrigger key={opt.value} value={opt.value} className={opt.value === "components" ? "inline-flex items-center gap-1.5" : undefined}>
                  {opt.value === "components" && <HammerIcon weight="duotone" className="size-3.5 text-violet-400" />}
                  {opt.label}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </div>
      )}

      {!query && wbmOnly && (
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <HammerIcon weight="duotone" className="size-4 text-violet-400 shrink-0" />
          <span className="text-muted-foreground shrink-0">Components:</span>
          <input
            type="number"
            inputMode="numeric"
            min={0}
            placeholder="Min"
            value={minInput}
            onChange={(e) => setMinInput(e.target.value)}
            className="h-8 w-20 rounded-md border border-input bg-transparent px-2 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
          />
          <span className="text-muted-foreground shrink-0">–</span>
          <input
            type="number"
            inputMode="numeric"
            min={0}
            placeholder="Max"
            value={maxInput}
            onChange={(e) => setMaxInput(e.target.value)}
            className="h-8 w-20 rounded-md border border-input bg-transparent px-2 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
          />
          {(minInput !== "" || maxInput !== "") && (
            <span className="flex items-center gap-1 text-xs text-muted-foreground" title="wbm-relay excludes any diagram NetEase's own feed didn't report a component count for, rather than guessing it's zero.">
              <InfoIcon weight="fill" className="size-3.5 shrink-0" />
              Some diagrams may be excluded — component count isn't known for every plan
            </span>
          )}
        </div>
      )}

      {error && (
        <p className="text-sm text-muted-foreground">
          Gallery temporarily unavailable ({error}). Try again shortly.
        </p>
      )}

      {!error && loading && (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="aspect-video rounded-xl" />
          ))}
        </div>
      )}

      {!error && !loading && !queryInvalid && plans.length === 0 && (
        <p className="text-sm text-muted-foreground">
          {activeSearch
            ? `No results for "${activeSearch}".`
            : wbmOnly
              ? "No WBM builders in this view yet."
              : "No builds in the gallery yet."}
        </p>
      )}

      {!error && !loading && plans.length > 0 && (
        <>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
            {plans.map((plan) => (
              <PlanCard
                key={plan.plan_id}
                plan={plan}
                wbmSlug={plan.author_number_id ? wbmBuilders[plan.author_number_id] : undefined}
                onWbmBadgeClick={() => setWbmOnly(true)}
              />
            ))}
          </div>
          {!activeSearch && (
            <div ref={sentinelRef} className="flex justify-center py-4 text-sm text-muted-foreground">
              {loadingMore && "Loading…"}
              {!loadingMore && !hasMore && "That's everything."}
            </div>
          )}
        </>
      )}
    </div>
  )
}
