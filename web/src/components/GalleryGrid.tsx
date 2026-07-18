import * as React from "react"
import { useEffect, useRef, useState } from "react"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { HammerIcon, HeartIcon, FireIcon, XIcon } from "@phosphor-icons/react"
import {
  WBM_RELAY_URL,
  SORT_OPTIONS,
  DEFAULT_SORT,
  CATEGORY_OPTIONS,
  planDetailUrl,
  formatCount,
  categoryLabel,
  type GalleryPlan,
  type PlanDetail,
} from "@/lib/gallery"

const LIMIT = 20

function StatRow({ plan, className = "" }: { plan: Pick<GalleryPlan, "heat_val" | "like_num" | "build_num">; className?: string }) {
  return (
    <div className={`flex items-center gap-3 ${className}`}>
      <span className="flex items-center gap-1">
        <FireIcon weight="fill" className="size-3.5 text-orange-400" /> {formatCount(plan.heat_val)}
      </span>
      <span className="flex items-center gap-1">
        <HeartIcon weight="fill" className="size-3.5 text-rose-400" /> {formatCount(plan.like_num)}
      </span>
      <span className="flex items-center gap-1">
        <HammerIcon weight="duotone" className="size-3.5" /> {formatCount(plan.build_num)}
      </span>
    </div>
  )
}

function DetailModal({ planId, onClose }: { planId: string; onClose: () => void }) {
  const [detail, setDetail] = useState<PlanDetail | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    document.body.style.overflow = "hidden"
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose()
      }
    }
    window.addEventListener("keydown", onKey)
    return () => {
      document.body.style.overflow = ""
      window.removeEventListener("keydown", onKey)
    }
  }, [onClose])

  useEffect(() => {
    setDetail(null)
    setError(null)
    fetch(planDetailUrl(planId))
      .then((res) => {
        if (!res.ok) {
          throw new Error(`relay returned ${res.status}`)
        }
        return res.json()
      })
      .then(setDetail)
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
  }, [planId])

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 backdrop-blur-sm p-4 sm:p-8"
      onClick={onClose}
    >
      <button
        onClick={onClose}
        aria-label="Close"
        className="absolute top-4 right-4 z-10 flex items-center justify-center size-9 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors cursor-pointer"
      >
        <XIcon weight="bold" className="size-5" />
      </button>

      <div
        className="relative w-full max-w-5xl max-h-full overflow-y-auto rounded-2xl bg-card ring-1 ring-white/10 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        {error && (
          <div className="p-8 text-center text-sm text-muted-foreground">
            Couldn't load this build ({error}).
          </div>
        )}

        {!error && !detail && (
          <div className="space-y-4 p-4">
            <Skeleton className="w-full aspect-video rounded-xl" />
            <Skeleton className="h-6 w-1/2" />
            <Skeleton className="h-4 w-full" />
          </div>
        )}

        {!error && detail && (
          <>
            <div className="relative">
              {detail.picture_url && (
                <img
                  src={detail.picture_url}
                  alt={detail.title}
                  className="w-full max-h-[65vh] object-contain bg-black"
                />
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
              {detail.description && (
                <p className="text-sm text-muted-foreground leading-relaxed whitespace-pre-line">
                  {detail.description}
                </p>
              )}
              <div className="flex flex-wrap gap-2">
                <Badge variant="outline" className="font-mono">{detail.art_code}</Badge>
                {detail.share_id && <Badge variant="outline" className="font-mono">{detail.share_id}</Badge>}
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  )
}

export function GalleryGrid() {
  const [sort, setSort] = useState<string>(DEFAULT_SORT)
  const [tag, setTag] = useState<number>(CATEGORY_OPTIONS[0].value)
  const [plans, setPlans] = useState<GalleryPlan[]>([])
  const [nextStart, setNextStart] = useState(0)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null)
  const [hasMore, setHasMore] = useState(true)
  const sentinelRef = useRef<HTMLDivElement | null>(null)

  // sort/tag change → reset and refetch page 0
  useEffect(() => {
    setLoading(true)
    setError(null)
    setHasMore(true)
    fetch(`${WBM_RELAY_URL}/api/gallery?sort=${sort}&tag=${tag}&start=0&limit=${LIMIT}`)
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
        setHasMore(fetched.length > 0)
      })
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false))
  }, [sort, tag])

  function loadMore() {
    setLoadingMore(true)
    fetch(`${WBM_RELAY_URL}/api/gallery?sort=${sort}&tag=${tag}&start=${nextStart}&limit=${LIMIT}`)
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

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap gap-1.5">
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
            {SORT_OPTIONS.map((opt) => (
              <TabsTrigger key={opt.value} value={opt.value}>{opt.label}</TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

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

      {!error && !loading && plans.length === 0 && (
        <p className="text-sm text-muted-foreground">No builds in the gallery yet.</p>
      )}

      {!error && !loading && plans.length > 0 && (
        <>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
            {plans.map((plan) => {
              const label = categoryLabel(plan.category_tag)
              return (
                <button
                  key={plan.plan_id}
                  onClick={() => setSelectedPlanId(plan.plan_id)}
                  className="group relative overflow-hidden rounded-xl ring-1 ring-border aspect-video bg-muted text-left cursor-pointer"
                >
                  {plan.picture_url && (
                    <img
                      src={plan.picture_url}
                      alt=""
                      loading="lazy"
                      className="absolute inset-0 h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                    />
                  )}
                  <div className="absolute inset-0 bg-linear-to-t from-black/70 to-transparent" />
                  {label && (
                    <span className="absolute top-2 left-2 text-[11px] font-medium px-2 py-0.5 rounded-md bg-black/60 text-white/90 backdrop-blur-sm">
                      {label}
                    </span>
                  )}
                  <div className="absolute bottom-0 left-0 right-0 p-3 flex items-end justify-between gap-2">
                    <StatRow plan={plan} className="text-xs text-white/90" />
                    {plan.author_name && (
                      <span className="text-sm text-white/80 truncate max-w-28 shrink-0">{plan.author_name}</span>
                    )}
                  </div>
                </button>
              )
            })}
          </div>
          <div ref={sentinelRef} className="flex justify-center py-4 text-sm text-muted-foreground">
            {loadingMore && "Loading…"}
            {!loadingMore && !hasMore && "That's everything."}
          </div>
        </>
      )}

      {selectedPlanId && (
        <DetailModal planId={selectedPlanId} onClose={() => setSelectedPlanId(null)} />
      )}
    </div>
  )
}
