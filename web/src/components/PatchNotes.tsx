import { useState, useMemo, useEffect } from "react"
import * as React from "react"
import { cn } from "@/lib/utils"
import { url } from "@/lib/url"

// Link to official patch notes article per version
const VERSION_LINKS: Record<string, string> = {
  "v1.7": url("/tutorials/patch-notes-may-28-2026#official-patch-notes"),
}

function isVideo(url: string) {
  const path = url.split("?")[0].toLowerCase()
  return path.endsWith(".mp4") || path.endsWith(".webm") || path.endsWith(".mov")
}

function isDiscordLink(url: string) {
  return url.startsWith("https://discord.com/channels/")
}

export interface Patch {
  coolness: "high" | "normal" | ""
  title: string
  description: string
  guild: boolean
  solo: boolean
  pc: boolean
  mobile: boolean
  ps5: boolean
  media: string[]
  version: string
  notes: string
}

const COOLNESS_STYLES: Record<string, string> = {
  high:   "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20",
  normal: "bg-sky-500/10 text-sky-600 dark:text-sky-400 border-sky-500/20",
}

const COOLNESS_LABEL: Record<string, string> = {
  high:   "Must know",
  normal: "Good to know",
}

const PLATFORM_STYLES: Record<string, string> = {
  guild:  "bg-violet-500/10 text-violet-700 dark:text-violet-300",
  solo:   "bg-sky-500/10 text-sky-700 dark:text-sky-300",
  pc:     "bg-orange-500/15 text-orange-700 dark:text-orange-300",
  mobile: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  ps5:    "bg-blue-500/10 text-blue-700 dark:text-blue-300",
}

const PLATFORM_LABEL: Record<string, string> = {
  guild: "Guild", solo: "Solo", pc: "PC", mobile: "Mobile", ps5: "PS5",
}

const MODE_PLATFORMS = ["guild", "solo"] as const
const DEVICE_PLATFORMS = ["pc", "mobile", "ps5"] as const
const PLATFORM_GROUPS = [
  { label: "Mode",     platforms: MODE_PLATFORMS },
  { label: "Platform", platforms: DEVICE_PLATFORMS },
] as const

// Parse "v1.7", "v1.10", "pre v1.7", "priori v1.7" → sortable float
function parseVersion(v: string): number {
  const m = v.match(/(\d+)\.(\d+)/)
  if (!m) return -1
  const base = parseInt(m[1]) * 1000 + parseInt(m[2])
  const isPre = /^(pre|priori)/i.test(v.trim())
  return isPre ? base - 0.5 : base
}

function Lightbox({ src, onClose }: { src: string | null; onClose: () => void }) {
  useEffect(() => {
    if (!src) return
    const handler = (e: KeyboardEvent) => { if (e.key === "Escape") onClose() }
    document.addEventListener("keydown", handler)
    document.body.style.overflow = "hidden"
    return () => {
      document.removeEventListener("keydown", handler)
      document.body.style.overflow = ""
    }
  }, [src, onClose])

  if (!src) return null
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/90 backdrop-blur-sm"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label="Media viewer"
    >
      <button
        onClick={onClose}
        className="absolute top-4 right-4 z-10 text-white/70 hover:text-white transition-colors p-2 rounded-full hover:bg-white/10"
        aria-label="Close"
      >
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
      <div className="flex flex-col items-center gap-2 p-6" onClick={(e) => e.stopPropagation()}>
        {isVideo(src) ? (
          <video src={src} className="max-h-[90vh] max-w-[90vw] rounded" controls autoPlay />
        ) : (
          <>
            <img src={src} alt="Screenshot" className="max-h-[90vh] max-w-[90vw] rounded object-contain" />
            <a
              href={src}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-white/40 hover:text-white/80 transition-colors"
            >
              Open full size ↗
            </a>
          </>
        )}
      </div>
    </div>
  )
}

const PLATFORMS = ["guild", "solo", "pc", "mobile", "ps5"] as const
type Platform = (typeof PLATFORMS)[number]
type CoolnessFilter = "all" | "high" | "normal"

function MediaThumb({ src, onClick }: { src: string; onClick: () => void }) {
  if (isDiscordLink(src)) {
    return (
      <a
        href={src}
        target="_blank"
        rel="noopener noreferrer"
        onClick={(e) => e.stopPropagation()}
        className="shrink-0 flex items-center justify-center h-10 w-14 rounded border border-border hover:border-[#5865F2]/60 bg-[#5865F2]/5 hover:bg-[#5865F2]/10 transition-colors"
        title="View on Discord"
        data-umami-event="patches_media_open"
      >
        <svg className="size-4 text-[#5865F2]" viewBox="0 0 24 24" fill="currentColor">
          <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028 14.09 14.09 0 0 0 1.226-1.994.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03z"/>
        </svg>
      </a>
    )
  }
  return (
    <button
      onClick={onClick}
      className="relative shrink-0 rounded overflow-hidden border border-border hover:border-primary transition-colors"
      data-umami-event="patches_media_open"
    >
      {isVideo(src) ? (
        <>
          <video src={src + "#t=0.1"} className="h-10 w-14 object-cover" preload="auto" muted />
          <span className="absolute inset-0 flex items-center justify-center bg-black/30">
            <svg className="size-3.5 text-white drop-shadow" viewBox="0 0 24 24" fill="currentColor">
              <path d="M8 5v14l11-7z"/>
            </svg>
          </span>
        </>
      ) : (
        <img src={src} alt="screenshot" className="h-10 w-14 object-cover" />
      )}
    </button>
  )
}

const COOLNESS_DOT: Record<string, string> = {
  high:   "bg-amber-500",
  normal: "bg-sky-500",
}

function PatchRow({ patch, onMedia }: { patch: Patch; onMedia: (src: string) => void }) {
  const platforms = [...MODE_PLATFORMS, ...DEVICE_PLATFORMS].filter((p) => patch[p])

  return (
    <div className="flex items-start gap-3 px-3 py-2.5 hover:bg-muted/30 transition-colors">
      <div
        title={COOLNESS_LABEL[patch.coolness] ?? ""}
        className={cn("mt-1.75 shrink-0 size-2 rounded-full", COOLNESS_DOT[patch.coolness] ?? "bg-border/50")}
      />

      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-foreground leading-snug">{patch.title}</p>
        {patch.description && (
          <p className="text-xs text-muted-foreground leading-relaxed mt-0.5">{patch.description}</p>
        )}
        {patch.notes && (
          <p className="text-xs text-amber-600 dark:text-amber-400 italic mt-0.5">{patch.notes}</p>
        )}
      </div>

      <div className="flex items-center gap-2 shrink-0 mt-0.5">
        <div className="flex flex-wrap justify-end gap-1">
          {platforms.map((p) => (
            <span key={p} className={cn("inline-block rounded px-1.5 py-0.5 text-[10px] font-medium", PLATFORM_STYLES[p])}>
              {PLATFORM_LABEL[p]}
            </span>
          ))}
        </div>
        {patch.media?.length > 0 && (
          <div className="flex gap-1">
            {patch.media.map((src, i) => (
              <MediaThumb key={i} src={src} onClick={() => onMedia(src)} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export function PatchNotes({ patches }: { patches: Patch[] }) {
  const [coolnessFilter, setCoolnessFilter] = useState<CoolnessFilter>("all")
  const [platformFilter, setPlatformFilter] = useState<Set<Platform>>(new Set())
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null)

  function togglePlatform(p: Platform) {
    setPlatformFilter((prev) => {
      const next = new Set(prev)
      if (next.has(p)) next.delete(p)
      else next.add(p)
      return next
    })
  }

  const { groups, totalVisible } = useMemo(() => {
    const filtered = patches.filter((p) => {
      if (!p.title) return false
      if (coolnessFilter !== "all" && p.coolness !== coolnessFilter) return false
      if (platformFilter.size > 0 && ![...platformFilter].some((pl) => p[pl])) return false
      return true
    })

    // Group by version
    const map = new Map<string, Patch[]>()
    for (const p of filtered) {
      const v = p.version || "Unknown"
      const arr = map.get(v) ?? []
      arr.push(p)
      map.set(v, arr)
    }

    // Sort versions descending
    const sorted = [...map.entries()].sort(
      ([a], [b]) => parseVersion(b) - parseVersion(a)
    )

    return { groups: sorted, totalVisible: filtered.length }
  }, [patches, coolnessFilter, platformFilter])

  const latestVersion = useMemo(() => {
    const versions = patches.map((p) => p.version).filter(Boolean)
    if (!versions.length) return null
    return versions.reduce((best, v) => parseVersion(v) > parseVersion(best) ? v : best)
  }, [patches])

  return (
    <div className="space-y-6">
      {/* Filter bar */}
      <div className="flex flex-wrap gap-2 items-center">
        <div className="flex gap-1.5 flex-wrap">
          {(["all", "high", "normal"] as const).map((c) => (
            <button
              key={c}
              onClick={() => setCoolnessFilter(c)}
              className={cn(
                "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
                coolnessFilter === c
                  ? c === "all"
                    ? "bg-foreground text-background border-foreground"
                    : cn(COOLNESS_STYLES[c], "opacity-100")
                  : "text-muted-foreground border-border hover:border-foreground/40",
              )}
            >
              {c === "all" ? "All" : COOLNESS_LABEL[c]}
            </button>
          ))}
        </div>
        {PLATFORM_GROUPS.map(({ label, platforms }) => (
          <React.Fragment key={label}>
            <div className="w-px h-4 bg-border" />
            <div className="flex items-center gap-1.5 flex-wrap">
              <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/50">{label}</span>
              {platforms.map((p) => (
                <button
                  key={p}
                  onClick={() => togglePlatform(p as Platform)}
                  className={cn(
                    "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
                    platformFilter.has(p as Platform)
                      ? cn(PLATFORM_STYLES[p], "border-current/30")
                      : "text-muted-foreground border-border hover:border-foreground/40",
                  )}
                >
                  {PLATFORM_LABEL[p]}
                </button>
              ))}
            </div>
          </React.Fragment>
        ))}
        <span className="text-xs text-muted-foreground ml-auto">
          {totalVisible} entr{totalVisible !== 1 ? "ies" : "y"}
        </span>
      </div>

      {/* Version groups */}
      {groups.length === 0 && (
        <p className="text-center text-muted-foreground py-12 text-sm">No entries match the current filter.</p>
      )}

      {groups.map(([version, items]) => (
        <div key={version} className="space-y-3">
          {/* Version header */}
          <div className="flex items-center gap-3">
            <h2 className="font-heading text-lg font-semibold">{version}</h2>
            {version === latestVersion && (
              <span className="inline-flex items-center rounded-full bg-primary/10 text-primary border border-primary/20 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide">
                Latest
              </span>
            )}
            {VERSION_LINKS[version] && (
              <a
                href={VERSION_LINKS[version]}
                className="text-xs text-muted-foreground hover:text-primary transition-colors underline underline-offset-2"
                data-umami-event="patches_official_notes_click"
                data-umami-event-version={version}
              >
                Official patch notes
              </a>
            )}
            <span className="text-xs text-muted-foreground">
              {items.length} {items.length === 1 ? "entry" : "entries"}
            </span>
            <div className="flex-1 h-px bg-border" />
          </div>

          {/* Patch list */}
          <div className="rounded-lg border border-border overflow-hidden divide-y divide-border/60">
            {items.map((patch, i) => (
              <PatchRow key={i} patch={patch} onMedia={setLightboxSrc} />
            ))}
          </div>
        </div>
      ))}

      <Lightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />
    </div>
  )
}
