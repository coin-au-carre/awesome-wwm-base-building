import * as React from "react"
import { useEffect, useState } from "react"
import { Skeleton } from "@/components/ui/skeleton"
import { ArrowLeftIcon, UserCircleIcon } from "@phosphor-icons/react"
import { DetailModal, StatRow, VisibilityBadge, CopyPill } from "@/components/GalleryGrid"
import { url } from "@/lib/url"
import {
  WBM_RELAY_URL,
  designerUrl,
  categoryLabel,
  isPrivate,
  type DesignerProfile,
  type DesignerPlan,
} from "@/lib/gallery"

function StatTile({ value, label }: { value: number | string; label: string }) {
  return (
    <div className="text-center">
      <div className="font-heading text-2xl sm:text-3xl font-bold italic">{value}</div>
      <div className="text-xs text-muted-foreground">{label}</div>
    </div>
  )
}

export function BuilderProfile() {
  const [numberId, setNumberId] = useState<string | null>(null)
  const [profile, setProfile] = useState<DesignerProfile | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [selectedPlan, setSelectedPlan] = useState<DesignerPlan | null>(null)

  // Query-string route (?id=<number_id>), not a dynamic [id] path
  // segment — the site builds statically with no build-time list of
  // designer ids. id is the public account number; wbm-relay resolves
  // whatever internal id it needs server-side (see gallery.ts's
  // designerUrl). See gallery/builder.astro.
  useEffect(() => {
    setNumberId(new URLSearchParams(location.search).get("id"))
  }, [])

  useEffect(() => {
    if (!numberId) return
    if (!WBM_RELAY_URL) {
      setError("not deployed yet")
      return
    }
    setError(null)
    fetch(designerUrl(numberId))
      .then((res) => {
        if (!res.ok) throw new Error(`relay returned ${res.status}`)
        return res.json()
      })
      .then(setProfile)
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
  }, [numberId])

  if (numberId === null) return null

  if (!numberId) {
    return <p className="text-sm text-muted-foreground">No builder specified.</p>
  }

  if (error) {
    return <p className="text-sm text-muted-foreground">Builder profile unavailable ({error}).</p>
  }

  if (!profile) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-20 w-full max-w-md rounded-xl" />
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="aspect-video rounded-xl" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <a
        href={url("/gallery")}
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeftIcon weight="bold" className="size-3.5" /> Back to gallery
      </a>
      <div className="flex flex-wrap items-center gap-6">
        <div className="flex items-center gap-3">
          <UserCircleIcon weight="fill" className="size-14 text-muted-foreground/50" />
          <h1 className="font-heading text-3xl font-bold leading-tight">{profile.nickname || profile.number_id}</h1>
        </div>
        <div className="flex items-center gap-6 ml-auto">
          <StatTile value={profile.follower_num} label="Fans" />
          <StatTile value={profile.like_num} label="Likes" />
          <StatTile value={profile.published_num} label="Published Works" />
        </div>
      </div>

      {profile.plans.length === 0 ? (
        <p className="text-sm text-muted-foreground">No published construction diagrams yet.</p>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
            <span>
              Showing all {profile.plans.length} published construction diagrams by {profile.nickname || profile.number_id}.
            </span>
            {profile.number_id && <CopyPill label="ID" value={profile.number_id} />}
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
            {profile.plans.map((plan) => {
              const label = categoryLabel(plan.category_tag)
              return (
                <button
                  key={plan.plan_id}
                  onClick={() => setSelectedPlan(plan)}
                  className="group relative overflow-hidden rounded-xl ring-1 ring-border aspect-video bg-muted text-left cursor-pointer"
                >
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
                  {isPrivate(plan.private) && <VisibilityBadge private_={plan.private} className="absolute top-2 right-2" />}
                  <div className="absolute bottom-0 left-0 right-0 p-3 flex items-end justify-end">
                    <StatRow plan={plan} className="text-xs text-white/90" />
                  </div>
                </button>
              )
            })}
          </div>
        </>
      )}

      {selectedPlan && (
        <DetailModal
          plan={{ plan_id: selectedPlan.plan_id, author_name: profile.nickname, author_number_id: profile.number_id }}
          onClose={() => setSelectedPlan(null)}
        />
      )}
    </div>
  )
}
