import * as React from "react"
import { useEffect, useState } from "react"
import { AvatarStatus, CopyPill } from "@/components/GalleryGrid"
import { InlineStat } from "@/components/BuildersDirectory"
import { BuilderExtraInfo } from "@/components/BuilderExtraInfo"
import { WBM_RELAY_URL, designerUrl, type DesignerProfile } from "@/lib/gallery"
import { UsersIcon, HeartIcon, StackIcon } from "@phosphor-icons/react"

// Same avatar/name/id/stats layout as the gallery profile header
// (BuilderProfile.tsx) — only fetches when this builder has a linked
// neteaseNumberId (see [slug].astro), otherwise falls back to the plain
// initials avatar with no live data. children renders below the name
// (alias line, contribution badges, tags) exactly as before.
export function BuilderProfileHeader({
  numberId,
  slug,
  displayName,
  initial,
  subtitle,
  children,
}: {
  numberId?: string
  // Keys the view-transition names shared with BuildersDirectory's
  // detail panel (builder-avatar-<slug>/builder-name-<slug>) — when a
  // visitor arrives here via "View full profile", Astro's ClientRouter
  // morphs the already-visible avatar/name into place instead of a full
  // crossfade, so the navigation reads as "more detail appeared" rather
  // than "new page loaded." Omit (e.g. arriving directly via URL, no
  // matching element on the previous page) and it's just a no-op.
  slug?: string
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
          <div style={slug ? { viewTransitionName: `builder-avatar-${slug}` } : undefined}>
            {profile?.avatar_url ? (
              <AvatarStatus
                src={profile.avatar_url}
                className="flex size-14 sm:size-24"
                level={profile.level}
                isOnline={profile.is_online}
              />
            ) : (
              <div className="flex size-14 sm:size-24 rounded-full bg-primary/10 text-primary items-center justify-center text-2xl sm:text-4xl font-bold shrink-0 ring-2 ring-primary/20">
                {initial}
              </div>
            )}
          </div>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 style={slug ? { viewTransitionName: `builder-name-${slug}` } : undefined} className="font-heading text-xl sm:text-3xl font-bold">{displayName}</h1>
              {profile?.number_id && <CopyPill label="ID" value={profile.number_id} />}
            </div>
            {subtitle && <p className="text-xs text-muted-foreground mt-0.5">{subtitle}</p>}
          </div>
        </div>
        {profile && (
          <div className="flex flex-wrap items-center gap-4 sm:ml-auto">
            <InlineStat icon={UsersIcon} value={profile.follower_num} label="Fans" className="text-blue-400" />
            <InlineStat icon={HeartIcon} value={profile.like_num} label="Likes" className="text-rose-400" />
            <InlineStat icon={StackIcon} value={profile.published_num} label="Published Works" className="text-amber-400" />
          </div>
        )}
      </div>
      {profile && <BuilderExtraInfo profile={profile} />}
      {children}
    </div>
  )
}
