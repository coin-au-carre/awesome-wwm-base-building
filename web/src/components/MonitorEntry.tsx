import * as React from "react"
import { useEffect, useState } from "react"
import { Avatar, CopyPill, builderHref } from "@/components/GalleryGrid"
import { BuilderGallerySection } from "@/components/BuilderGallerySection"
import { WBM_RELAY_URL, designerUrl, type DesignerProfile } from "@/lib/gallery"

const RING = {
  red: "ring-rose-500/60",
  indigo: "ring-indigo-500/60",
  blue: "ring-blue-500/60",
  purple: "ring-purple-500/60",
} as const

// One watched designer's card: avatar/nickname header (fetched directly,
// same call BuilderGallerySection makes internally for its own grid —
// two requests per entry, acceptable for a short hand-picked watchlist,
// not worth plumbing a shared fetch for a handful of entries) plus their
// full diagram grid, framed in the group's color. See copyright-watch.astro.
export function MonitorEntry({ numberId, color }: { numberId: string; color: keyof typeof RING }) {
  const [profile, setProfile] = useState<DesignerProfile | null>(null)

  useEffect(() => {
    if (!WBM_RELAY_URL) return
    fetch(designerUrl(numberId))
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => data && setProfile(data))
      .catch(() => {})
  }, [numberId])

  return (
    <div className={`rounded-2xl ring-2 ${RING[color]} bg-card p-4 space-y-3`}>
      <div className="flex items-center gap-3">
        <Avatar src={profile?.avatar_url} className="size-12" />
        <div className="min-w-0">
          <a href={builderHref(numberId)} className="font-heading font-semibold hover:text-primary transition-colors truncate block">
            {profile?.nickname || numberId}
          </a>
          <CopyPill label="ID" value={numberId} />
        </div>
      </div>
      <BuilderGallerySection numberId={numberId} />
    </div>
  )
}
