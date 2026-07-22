import * as React from "react"
import { useEffect, useState } from "react"
import { Avatar, CopyPill, builderHref } from "@/components/GalleryGrid"
import { BuilderGallerySection } from "@/components/BuilderGallerySection"
import { WBM_RELAY_URL, designerUrl, modPlayerUrl, type DesignerProfile, type ModPlayerDetail } from "@/lib/gallery"
import { readModKey, MOD_KEY_EVENT } from "@/components/ModKeyGate"

const RING = {
  red: "ring-rose-500/60",
  indigo: "ring-indigo-500/60",
  blue: "ring-blue-500/60",
  purple: "ring-purple-500/60",
  orange: "ring-orange-500/60",
} as const

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
        <div className="rounded-lg border border-rose-500/40 bg-rose-500/5 p-3 space-y-1.5 text-sm">
          <p className="font-semibold text-rose-500">Mod-only detail</p>
          <p>
            Level {modDetail.level}
            {modDetail.oversea_tag && ` · ${modDetail.oversea_tag}`}
            {" · "}
            {modDetail.is_online ? "Online" : "Offline"}
          </p>
          {modDetail.bio && <p className="text-muted-foreground whitespace-pre-wrap wrap-break-word">"{modDetail.bio}"</p>}
          {(modDetail.discord_account_id || modDetail.discord_global_name) && (
            <p>Discord: {modDetail.discord_global_name || modDetail.discord_account_id}</p>
          )}
          {modDetail.steam_account_id && <p>Steam: {modDetail.steam_account_id}</p>}
          {(modDetail.xbox_account_id || modDetail.xbox_username) && (
            <p>Xbox: {modDetail.xbox_username || modDetail.xbox_account_id}</p>
          )}
          {modDetail.psn_user_name && <p>PSN: {modDetail.psn_user_name}</p>}
          {modDetail.home_spaces && modDetail.home_spaces.length > 0 && (
            <div className="flex flex-wrap gap-1.5 pt-1">
              {modDetail.home_spaces.map((space) => (
                <CopyPill key={space.space_id} label={`Home (Lv.${space.level})`} value={space.space_id} />
              ))}
            </div>
          )}
        </div>
      )}
      {!failed && <BuilderGallerySection numberId={numberId} profile={profile} />}
    </div>
  )
}
