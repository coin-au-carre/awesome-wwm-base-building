import * as React from "react"
import { useEffect, useState } from "react"
import { Avatar, CopyPill, builderHref, renderChatText } from "@/components/GalleryGrid"
import { BuilderGallerySection } from "@/components/BuilderGallerySection"
import { WBM_RELAY_URL, designerUrl, modPlayerUrl, deviceLabel, type DesignerProfile, type ModPlayerDetail } from "@/lib/gallery"
import { readModKey, MOD_KEY_EVENT } from "@/components/ModKeyGate"
import { formatUnixSeconds } from "@/lib/dates"

const RING = {
  red: "ring-rose-500/60",
  indigo: "ring-indigo-500/60",
  blue: "ring-blue-500/60",
  purple: "ring-purple-500/60",
  orange: "ring-orange-500/60",
  green: "ring-green-500/60",
} as const

// seconds -> "12d 4h" (cumulative playtime, not a point in time).
function formatDuration(seconds: number | undefined): string | null {
  if (!seconds) return null
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  if (days > 0) return `${days}d ${hours}h`
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

function Stat({ label, value, title }: { label: string; value: React.ReactNode; title?: string }) {
  return (
    <div title={title}>
      <div className="text-[11px] uppercase tracking-wide text-muted-foreground leading-none">{label}</div>
      <div className="mt-0.5">{value}</div>
    </div>
  )
}

// One watched designer's card: avatar/nickname header + their full
// diagram grid, framed in the group's color. Fetches designerUrl(numberId)
// once here and passes the result into BuilderGallerySection (which would
// otherwise redundantly fetch the exact same endpoint itself) — see its
// providedProfile doc comment. See copyright-watch.astro.
export function MonitorEntry({
  numberId,
  color,
  note,
  diagrams,
}: {
  numberId: string
  color: keyof typeof RING
  note?: string
  diagrams?: string[]
}) {
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

  // Mod-only detail (bio, linked accounts, home spaces) — only fetched
  // when a mod key is present in this tab (see ModKeyGate). A wrong/no
  // key just 401s server-side; modDetail stays null and nothing extra
  // renders, same fail-silent pattern as the public profile fetch above.
  const [modDetail, setModDetail] = useState<ModPlayerDetail | null>(null)

  useEffect(() => {
    function loadModDetail() {
      const key = readModKey()
      if (!WBM_RELAY_URL || !key) {
        setModDetail(null)
        return
      }
      fetch(modPlayerUrl(numberId, key))
        .then((res) => {
          if (!res.ok) throw new Error(`relay returned ${res.status}`)
          return res.json()
        })
        .then((data) => setModDetail(data))
        .catch(() => setModDetail(null))
    }

    loadModDetail()
    window.addEventListener(MOD_KEY_EVENT, loadModDetail)
    return () => window.removeEventListener(MOD_KEY_EVENT, loadModDetail)
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
      {note && <p className="text-sm text-muted-foreground">{note}</p>}
      {diagrams && diagrams.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {diagrams.map((code) => (
            <CopyPill key={code} label="Diagram" value={code} />
          ))}
        </div>
      )}
      {modDetail && (
        <div className="rounded-lg border border-border bg-muted/30 p-3 space-y-3 text-sm">
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-3 gap-y-2">
            <Stat label="Level" value={modDetail.level} />
            <Stat label="Region" value={modDetail.oversea_tag || "—"} />
            <Stat
              label="Status"
              value={
                <span className={modDetail.is_online ? "text-emerald-500" : "text-muted-foreground"}>
                  {modDetail.is_online ? "Online" : "Offline"}
                </span>
              }
            />
            {formatUnixSeconds(modDetail.create_time) && (
              <Stat label="Joined" value={formatUnixSeconds(modDetail.create_time)!.relative} title={formatUnixSeconds(modDetail.create_time)!.full} />
            )}
            {formatUnixSeconds(modDetail.login_time) && (
              <Stat label="Last login" value={formatUnixSeconds(modDetail.login_time)!.relative} title={formatUnixSeconds(modDetail.login_time)!.full} />
            )}
            {formatUnixSeconds(modDetail.logout_time) && (
              <Stat label="Last logout" value={formatUnixSeconds(modDetail.logout_time)!.relative} title={formatUnixSeconds(modDetail.logout_time)!.full} />
            )}
            {formatDuration(modDetail.online_time) && <Stat label="Playtime" value={formatDuration(modDetail.online_time)} />}
            {modDetail.device_name && <Stat label="Device" value={deviceLabel(modDetail.device_name)} />}
            {!!modDetail.max_xiuwei_kungfu && <Stat label="Power" value={modDetail.max_xiuwei_kungfu.toLocaleString()} />}
          </div>

          {modDetail.bio && (
            <p className="italic text-muted-foreground whitespace-pre-wrap wrap-break-word border-l-2 border-border pl-2">
              {renderChatText(modDetail.bio)}
            </p>
          )}

          {(modDetail.discord_account_id || modDetail.discord_global_name || modDetail.steam_account_id || modDetail.xbox_account_id || modDetail.xbox_username || modDetail.psn_user_name) && (
            <div className="flex flex-wrap gap-x-4 gap-y-1">
              {(modDetail.discord_account_id || modDetail.discord_global_name) && (
                <span><span className="text-muted-foreground">Discord:</span> {modDetail.discord_global_name || modDetail.discord_account_id}</span>
              )}
              {modDetail.steam_account_id && <span><span className="text-muted-foreground">Steam:</span> {modDetail.steam_account_id}</span>}
              {(modDetail.xbox_account_id || modDetail.xbox_username) && (
                <span><span className="text-muted-foreground">Xbox:</span> {modDetail.xbox_username || modDetail.xbox_account_id}</span>
              )}
              {modDetail.psn_user_name && <span><span className="text-muted-foreground">PSN:</span> {modDetail.psn_user_name}</span>}
            </div>
          )}

          {modDetail.home_spaces && modDetail.home_spaces.length > 0 && (
            <div className="space-y-1">
              <p className="text-[11px] uppercase tracking-wide text-muted-foreground leading-none">Homestead</p>
              <div className="flex flex-wrap gap-1.5">
                {modDetail.home_spaces.map((space) => (
                  <CopyPill key={space.space_id} label={`Lv.${space.level}`} value={space.space_id} />
                ))}
              </div>
            </div>
          )}

          {modDetail.home_works && modDetail.home_works.length > 0 && (
            <div className="space-y-1">
              <p className="text-[11px] uppercase tracking-wide text-muted-foreground leading-none">
                Showcased works ({modDetail.home_works.length})
              </p>
              <div className="flex flex-wrap gap-1.5">
                {modDetail.home_works.map((work) => (
                  <CopyPill key={work.work_id} label={`Type ${work.work_type}`} value={work.work_id} />
                ))}
              </div>
            </div>
          )}

          {(modDetail.campaign_slogan || modDetail.campaign_banner_url) && (
            <div className="flex items-center gap-2">
              {modDetail.campaign_banner_url && (
                <img src={modDetail.campaign_banner_url} alt="Campaign banner" className="h-8 w-14 rounded object-cover shrink-0" />
              )}
              {modDetail.campaign_slogan && <p className="italic text-muted-foreground">{renderChatText(modDetail.campaign_slogan)}</p>}
            </div>
          )}
        </div>
      )}
      {!failed && <BuilderGallerySection numberId={numberId} profile={profile} />}
    </div>
  )
}
