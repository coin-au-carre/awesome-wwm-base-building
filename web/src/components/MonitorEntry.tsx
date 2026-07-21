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

// One watched designer's card: avatar/nickname header + their full
// diagram grid, framed in the group's color. Fetches designerUrl(numberId)
// once here and passes the result into BuilderGallerySection (which would
// otherwise redundantly fetch the exact same endpoint itself) — see its
// providedProfile doc comment. See copyright-watch.astro.
export function MonitorEntry({ numberId, color }: { numberId: string; color: keyof typeof RING }) {
  const [profile, setProfile] = useState<DesignerProfile | null>(null)
  // profile alone can't distinguish "still loading" from "failed" (both
  // null) — without this, a failed fetch would leave BuilderGallerySection
  // reading its passed-in null as "loading" forever instead of "give up."
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (!WBM_RELAY_URL) {
      setFailed(true)
      return
    }
    fetch(designerUrl(numberId))
      .then((res) => {
        if (!res.ok) throw new Error(`relay returned ${res.status}`)
        return res.json()
      })
      .then((data) => setProfile(data))
      .catch(() => setFailed(true))
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
      {!failed && <BuilderGallerySection numberId={numberId} profile={profile} />}
    </div>
  )
}
