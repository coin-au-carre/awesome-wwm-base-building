import * as React from "react"
import { useEffect, useState } from "react"
import { Skeleton } from "@/components/ui/skeleton"
import { PlanCard } from "@/components/GalleryGrid"
import { WBM_RELAY_URL, designerUrl, type DesignerProfile } from "@/lib/gallery"
import { url } from "@/lib/url"

const PREVIEW_COUNT = 8

// Embeds a preview of a builder's live NetEase gallery diagrams on their
// WBM profile page (/builders/<slug>) — only rendered when
// data/builder_identities.json links this builder to a neteaseNumberId.
// See docs/builder-identity.md's "merging the two builder-profile
// systems": this is the WBM-page half of that merge, the reverse
// direction (a banner-link back to here) lives in BuilderProfile.tsx.
// Fails quietly (renders nothing) on error — this is a bonus section on
// an otherwise-complete profile page, not core content worth an error
// message of its own.
export function BuilderGallerySection({ numberId }: { numberId: string }) {
  const [profile, setProfile] = useState<DesignerProfile | null>(null)
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

  if (failed || (profile && profile.plans.length === 0)) return null

  if (!profile) {
    return (
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="aspect-video rounded-xl" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        {profile.plans.slice(0, PREVIEW_COUNT).map((plan) => (
          <PlanCard key={plan.plan_id} plan={plan} showAuthor={false} />
        ))}
      </div>
      {profile.plans.length > PREVIEW_COUNT && (
        <a
          href={url(`/gallery/builder?id=${encodeURIComponent(numberId)}`)}
          className="inline-block text-xs text-muted-foreground hover:text-foreground transition-colors"
          data-umami-event="builder_profile_gallery_view_all_click"
        >
          View all {profile.plans.length} diagrams →
        </a>
      )}
    </div>
  )
}
