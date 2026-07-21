import * as React from "react"
import { useEffect, useState } from "react"
import { Skeleton } from "@/components/ui/skeleton"
import { PlanCard } from "@/components/GalleryGrid"
import { WBM_RELAY_URL, designerUrl, type DesignerProfile } from "@/lib/gallery"

// Embeds every one of a builder's live NetEase gallery diagrams on their
// WBM profile page (/builders/<slug>) — only rendered when
// data/builder_identities.json links this builder to a neteaseNumberId.
// Fan/like/published stats + id live in the page header instead (see
// BuilderProfileHeader.tsx), this just renders the diagram grid. See
// docs/builder-identity.md's "merging the two builder-profile systems":
// this is the WBM-page half of that merge, the reverse direction is a
// straight redirect (see BuilderProfile.tsx's wbmSlugs handling) since a
// WBM-linked builder now only ever has this one page. Fails quietly
// (renders nothing) on error — this is a bonus section on an otherwise-
// complete profile page, not core content worth an error message of its
// own.
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

  if (failed || (profile && profile.plans.length === 0)) { return null }

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
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
      {profile.plans.map((plan) => (
        <PlanCard key={plan.plan_id} plan={plan} showAuthor={false} />
      ))}
    </div>
  )
}
