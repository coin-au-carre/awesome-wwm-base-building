import * as React from "react"
import { useEffect, useState } from "react"
import { Skeleton } from "@/components/ui/skeleton"
import { PlanDetailContent, ShareButton } from "@/components/GalleryGrid"
import { BackLink, GalleryLink } from "@/components/BackLink"
import { WBM_RELAY_URL, planDetailUrl, type PlanDetail } from "@/lib/gallery"

export function PlanPage() {
  const [shareCode, setShareCode] = useState<string | null>(null)
  const [detail, setDetail] = useState<PlanDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [notFound, setNotFound] = useState(false)

  // Query-string route (?share=<SHARE code>), not a dynamic [id] path
  // segment — same reasoning as BuilderProfile. See gallery/plan.astro.
  useEffect(() => {
    setShareCode(new URLSearchParams(location.search).get("share"))
  }, [])

  useEffect(() => {
    if (!shareCode) return
    if (!WBM_RELAY_URL) {
      setError("not deployed yet")
      return
    }
    setError(null)
    setNotFound(false)
    fetch(planDetailUrl(shareCode))
      .then((res) => {
        if (res.status === 404) {
          setNotFound(true)
          return null
        }
        if (!res.ok) throw new Error(`relay returned ${res.status}`)
        return res.json()
      })
      .then((data) => data && setDetail(data))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
  }, [shareCode])

  // The browser tab/bookmark should show the diagram's own name, not
  // the generic page title — this is the one piece of "shareable link"
  // polish a static Astro page can still do purely client-side.
  useEffect(() => {
    if (detail?.title) {
      document.title = `${detail.title} | Where Builders Meet`
    }
  }, [detail])

  if (shareCode === null) return null

  if (!shareCode) {
    return <p className="text-sm text-muted-foreground">No diagram specified.</p>
  }

  if (notFound) {
    return (
      <div className="space-y-4">
        <div className="flex gap-2">
          <BackLink />
          <GalleryLink />
        </div>
        <p className="text-sm text-muted-foreground">
          No diagram found for share code "{shareCode}". Double-check the code.
        </p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-4">
        <div className="flex gap-2">
          <BackLink />
          <GalleryLink />
        </div>
        <p className="text-sm text-muted-foreground">Diagram unavailable ({error}).</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <BackLink />
        <GalleryLink />
        {detail && <ShareButton label="Share diagram page" />}
      </div>

      {!detail ? (
        <div className="max-w-3xl mx-auto space-y-4">
          <Skeleton className="w-full aspect-video rounded-xl" />
          <Skeleton className="h-6 w-1/2" />
          <Skeleton className="h-4 w-full" />
        </div>
      ) : (
        <div className="max-w-3xl mx-auto overflow-hidden rounded-2xl bg-card ring-1 ring-border shadow-sm">
          <PlanDetailContent detail={detail} />
        </div>
      )}
    </div>
  )
}
