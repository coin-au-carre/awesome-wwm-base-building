import * as React from "react"
import { useState } from "react"
import { GlobeIcon, DeviceMobileIcon, SwordIcon, ClockIcon } from "@phosphor-icons/react"
import { CopyPill, renderChatText } from "@/components/GalleryGrid"
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog"
import { deviceLabel, type DesignerProfile } from "@/lib/gallery"
import { formatUnixSeconds } from "@/lib/dates"

// Same colored icon-pill language as the builders directory's list-row
// tags (Region/Device/Martial Mastery) — one line, icon-first, instead
// of stacked label-over-value tiles. Kept local to this file rather than
// importing BuildersDirectory's Tag, to avoid coupling two otherwise
// independent components together over one small shared visual pattern.
function Pill({ children, className, title }: { children: React.ReactNode; className: string; title?: string }) {
  return (
    <span title={title} className={`inline-flex items-center gap-1 text-xs font-medium rounded-full px-2.5 py-1 ${className}`}>
      {children}
    </span>
  )
}

// Region/device/Martial Mastery/signature + last-seen/logged-in-since +
// campaign slogan/banner + showcased works — shared between the
// builders directory's detail panel and the full builder profile page,
// since both show the same DesignerProfile fields. compact drops the
// campaign/showcased-works sections for the directory's tighter side
// panel; the full profile passes compact=false to show everything.
export function BuilderExtraInfo({ profile, compact = false }: { profile: DesignerProfile; compact?: boolean }) {
  const [bannerOpen, setBannerOpen] = useState(false)
  // Last seen only makes sense while offline (logout_time is when they
  // *stopped* being online); logged-in-since only while online
  // (login_time of the still-active session) — showing both regardless
  // of is_online would be misleading once one of them goes stale.
  const lastSeen = !profile.is_online ? formatUnixSeconds(profile.logout_time) : null
  const loggedInSince = profile.is_online ? formatUnixSeconds(profile.login_time) : null
  const hasStats = profile.oversea_tag || profile.device_name || profile.max_xiuwei_kungfu || lastSeen || loggedInSince
  return (
    <div className="space-y-3">
      {hasStats && (
        <div className="flex flex-wrap gap-1.5">
          {profile.oversea_tag && (
            <Pill className="bg-sky-500/10 text-sky-600 dark:text-sky-300">
              <GlobeIcon weight="fill" className="size-3.5" />
              {profile.oversea_tag}
            </Pill>
          )}
          {profile.device_name && (
            <Pill className="bg-slate-500/10 text-slate-600 dark:text-slate-300">
              <DeviceMobileIcon weight="fill" className="size-3.5" />
              {deviceLabel(profile.device_name)}
            </Pill>
          )}
          {!!profile.max_xiuwei_kungfu && (
            <Pill className="bg-amber-500/10 text-white" title="Martial Mastery">
              <SwordIcon weight="fill" className="size-3.5" />
              {profile.max_xiuwei_kungfu.toLocaleString()}
            </Pill>
          )}
          {loggedInSince && (
            <Pill className="bg-emerald-500/10 text-emerald-600 dark:text-emerald-300" title={loggedInSince.full}>
              <ClockIcon weight="fill" className="size-3.5" />
              Online since {loggedInSince.relative}
            </Pill>
          )}
          {lastSeen && (
            <Pill className="bg-muted text-muted-foreground" title={lastSeen.full}>
              <ClockIcon weight="fill" className="size-3.5" />
              Last seen {lastSeen.relative}
            </Pill>
          )}
        </div>
      )}

      {profile.bio && (
        <p className="text-base italic text-muted-foreground whitespace-pre-wrap wrap-break-word border-l-2 border-border pl-2">
          {renderChatText(profile.bio)}
        </p>
      )}

      {!compact && (profile.campaign_slogan || profile.campaign_banner_url) && (
        <div className="space-y-2">
          {profile.campaign_slogan && <p className="text-sm italic text-muted-foreground">{renderChatText(profile.campaign_slogan)}</p>}
          {profile.campaign_banner_url && (
            <>
              <button
                type="button"
                onClick={() => setBannerOpen(true)}
                className="block cursor-zoom-in"
                title="Click to enlarge"
              >
                <img
                  src={profile.campaign_banner_url}
                  alt="Campaign banner"
                  className="h-32 w-full max-w-xs rounded-lg object-cover transition-opacity hover:opacity-90"
                />
              </button>
              <Dialog open={bannerOpen} onOpenChange={setBannerOpen}>
                <DialogContent className="max-w-3xl border-none bg-transparent p-2 shadow-none">
                  <DialogTitle className="sr-only">Campaign banner</DialogTitle>
                  <img src={profile.campaign_banner_url} alt="Campaign banner" className="h-auto w-full rounded-lg" />
                </DialogContent>
              </Dialog>
            </>
          )}
        </div>
      )}

      {!compact && profile.home_works && profile.home_works.length > 0 && (
        <div className="space-y-1.5">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Showcased Works ({profile.home_works.length})
          </h3>
          <div className="flex flex-wrap gap-1.5">
            {profile.home_works.map((work) => (
              <CopyPill key={work.work_id} label={`Type ${work.work_type}`} value={work.work_id} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
