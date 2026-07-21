import * as React from "react"
import { useEffect, useState } from "react"
import { Avatar, CopyPill } from "@/components/GalleryGrid"
import { StatTile } from "@/components/BuilderProfile"
import { WBM_RELAY_URL, designerUrl, type DesignerProfile } from "@/lib/gallery"

// Same avatar/name/id/stats layout as the gallery profile header
// (BuilderProfile.tsx) — only fetches when this builder has a linked
// neteaseNumberId (see [slug].astro), otherwise falls back to the plain
// initials avatar with no live data. children renders below the name
// (alias line, contribution badges, tags) exactly as before.
export function BuilderProfileHeader({
  numberId,
  displayName,
  initial,
  subtitle,
  children,
}: {
  numberId?: string
  displayName: string
  initial: string
  // Rendered as one line directly under the name (Discord name / aliases)
  // — separate from children, which renders below the whole avatar/name/
  // stats row (contribution badges), not tucked under just the name.
  subtitle?: React.ReactNode
  children?: React.ReactNode
}) {
  const [profile, setProfile] = useState<DesignerProfile | null>(null)

  useEffect(() => {
    if (!numberId || !WBM_RELAY_URL) { return }
    fetch(designerUrl(numberId))
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => data && setProfile(data))
      .catch(() => {})
  }, [numberId])

  return (
    <div className="space-y-4 rounded-2xl ring-1 ring-border bg-card p-5 sm:p-6">
      <div className="flex flex-wrap items-center gap-6">
        <div className="flex items-center gap-5">
          {profile?.avatar_url ? (
            <Avatar src={profile.avatar_url} className="flex size-14 sm:size-24" />
          ) : (
            <div className="flex size-14 sm:size-24 rounded-full bg-primary/10 text-primary items-center justify-center text-2xl sm:text-4xl font-bold shrink-0 ring-2 ring-primary/20">
              {initial}
            </div>
          )}
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="font-heading text-xl sm:text-3xl font-bold">{displayName}</h1>
              {profile?.number_id && <CopyPill label="ID" value={profile.number_id} />}
            </div>
            {subtitle && <p className="text-xs text-muted-foreground mt-0.5">{subtitle}</p>}
          </div>
        </div>
        {profile && (
          <div className="flex items-center gap-6 sm:ml-auto">
            <StatTile value={profile.follower_num} label="Fans" />
            <StatTile value={profile.like_num} label="Likes" />
            <StatTile value={profile.published_num} label="Published Works" />
          </div>
        )}
      </div>
      {children}
    </div>
  )
}
