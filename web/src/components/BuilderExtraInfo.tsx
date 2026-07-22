import * as React from "react"
import { CopyPill, renderChatText } from "@/components/GalleryGrid"
import { deviceLabel, type DesignerProfile } from "@/lib/gallery"

function Stat({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div>
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground leading-none">{label}</div>
      <div className="mt-0.5 text-sm">{value}</div>
    </div>
  )
}

// Region/device/Martial Mastery/signature + campaign slogan/banner +
// showcased works — shared between the builders directory's detail
// panel and the full builder profile page, since both show the same
// DesignerProfile fields. compact drops the campaign/showcased-works
// sections for the directory's tighter side panel; the full profile
// passes compact=false to show everything.
export function BuilderExtraInfo({ profile, compact = false }: { profile: DesignerProfile; compact?: boolean }) {
  const hasStats = profile.oversea_tag || profile.device_name || profile.max_xiuwei_kungfu
  return (
    <div className="space-y-3">
      {hasStats && (
        <div className="grid grid-cols-3 gap-3">
          {profile.oversea_tag && <Stat label="Region" value={profile.oversea_tag} />}
          {profile.device_name && <Stat label="Device" value={deviceLabel(profile.device_name)} />}
          {!!profile.max_xiuwei_kungfu && <Stat label="Martial Mastery" value={profile.max_xiuwei_kungfu.toLocaleString()} />}
        </div>
      )}

      {profile.bio && (
        <p className="text-sm italic text-muted-foreground whitespace-pre-wrap wrap-break-word border-l-2 border-border pl-2">
          {renderChatText(profile.bio)}
        </p>
      )}

      {!compact && (profile.campaign_slogan || profile.campaign_banner_url) && (
        <div className="flex items-center gap-2">
          {profile.campaign_banner_url && (
            <img src={profile.campaign_banner_url} alt="Campaign banner" className="h-10 w-16 rounded object-cover shrink-0" />
          )}
          {profile.campaign_slogan && <p className="text-sm italic text-muted-foreground">{renderChatText(profile.campaign_slogan)}</p>}
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
