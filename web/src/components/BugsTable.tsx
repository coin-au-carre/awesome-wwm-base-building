import { useState, useMemo, useEffect } from "react"
import * as React from "react"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { cn } from "@/lib/utils"

function isVideo(url: string) {
  const path = url.split("?")[0].toLowerCase()
  return path.endsWith(".mp4") || path.endsWith(".webm") || path.endsWith(".mov")
}

function isDiscordLink(url: string) {
  return url.startsWith("https://discord.com/channels/")
}

export interface Bug {
  severity: "high" | "normal" | "low" | "fixed" | ""
  title: string
  details: string
  guild: boolean
  solo: boolean
  pc: boolean
  mobile: boolean
  ps5: boolean
  media: string[]
  version: string
  date: string
  notes: string
}

const SEVERITY_STYLES: Record<string, string> = {
  high:   "bg-red-500/15 text-red-600 dark:text-red-400 border-red-500/30",
  normal: "bg-yellow-500/15 text-yellow-600 dark:text-yellow-400 border-yellow-500/30",
  low:    "bg-slate-500/10 text-slate-600 dark:text-slate-400 border-slate-500/20",
  fixed:  "bg-green-500/15 text-green-600 dark:text-green-400 border-green-500/30",
}

const SEVERITY_STYLES_INACTIVE: Record<string, string> = {
  high:   "bg-red-500/5 text-red-600/60 dark:text-red-400/50 border-red-500/15 hover:bg-red-500/10 hover:text-red-600 dark:hover:text-red-400 hover:border-red-500/30",
  normal: "bg-yellow-500/5 text-yellow-600/60 dark:text-yellow-400/50 border-yellow-500/15 hover:bg-yellow-500/10 hover:text-yellow-600 dark:hover:text-yellow-400 hover:border-yellow-500/30",
  low:    "bg-slate-500/5 text-slate-600/60 dark:text-slate-400/50 border-slate-500/15 hover:bg-slate-500/10 hover:text-slate-600 dark:hover:text-slate-400 hover:border-slate-500/30",
  fixed:  "bg-green-500/5 text-green-600/60 dark:text-green-400/50 border-green-500/15 hover:bg-green-500/10 hover:text-green-600 dark:hover:text-green-400 hover:border-green-500/30",
}

const PLATFORM_STYLES: Record<string, string> = {
  guild:  "bg-violet-500/15 text-violet-700 dark:text-violet-300",
  solo:   "bg-rose-500/15 text-rose-700 dark:text-rose-300",
  pc:     "bg-orange-500/15 text-orange-700 dark:text-orange-300",
  mobile: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  ps5:    "bg-blue-500/15 text-blue-700 dark:text-blue-300",
}

const PLATFORM_STYLES_INACTIVE: Record<string, string> = {
  guild:  "bg-violet-500/5 text-violet-600/60 dark:text-violet-400/50 border-violet-500/20 hover:bg-violet-500/10 hover:text-violet-700 dark:hover:text-violet-300 hover:border-violet-500/35",
  solo:   "bg-rose-500/5 text-rose-600/60 dark:text-rose-400/50 border-rose-500/20 hover:bg-rose-500/10 hover:text-rose-700 dark:hover:text-rose-300 hover:border-rose-500/35",
  pc:     "bg-orange-500/5 text-orange-600/60 dark:text-orange-400/50 border-orange-500/20 hover:bg-orange-500/10 hover:text-orange-700 dark:hover:text-orange-300 hover:border-orange-500/35",
  mobile: "bg-emerald-500/5 text-emerald-600/60 dark:text-emerald-400/50 border-emerald-500/20 hover:bg-emerald-500/10 hover:text-emerald-700 dark:hover:text-emerald-300 hover:border-emerald-500/35",
  ps5:    "bg-blue-500/5 text-blue-600/60 dark:text-blue-400/50 border-blue-500/20 hover:bg-blue-500/10 hover:text-blue-700 dark:hover:text-blue-300 hover:border-blue-500/35",
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
          <video
            src={src}
            className="max-h-[90vh] max-w-[90vw] rounded"
            controls
            autoPlay
          />
        ) : (
          <>
            <img
              src={src}
              alt="Screenshot"
              className="max-h-[90vh] max-w-[90vw] rounded object-contain"
            />
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

const SEVERITY_ORDER: Record<string, number> = { high: 0, normal: 1, low: 2, fixed: 3 }

const PLATFORMS = ["guild", "solo", "pc", "mobile", "ps5"] as const
type Platform = (typeof PLATFORMS)[number]

type SeverityFilter = "all" | "high" | "normal" | "low" | "fixed"
type SortDir = "asc" | "desc"

export function BugsTable({ bugs }: { bugs: Bug[] }) {
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>("all")
  const [platformFilter, setPlatformFilter] = useState<Set<Platform>>(new Set())
  const [severitySort, setSeveritySort] = useState<SortDir>("asc")
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null)

  function togglePlatform(p: Platform) {
    setPlatformFilter((prev) => {
      const next = new Set(prev)
      if (next.has(p)) next.delete(p)
      else next.add(p)
      return next
    })
  }

  const filtered = useMemo(() => {
    const result = bugs.filter((bug) => {
      if (!bug.title) return false
      if (severityFilter !== "all" && bug.severity !== severityFilter) return false
      if (platformFilter.size > 0 && ![...platformFilter].some((p) => bug[p])) return false
      return true
    })
    result.sort((a, b) => {
      const diff = (SEVERITY_ORDER[a.severity] ?? 99) - (SEVERITY_ORDER[b.severity] ?? 99)
      return severitySort === "asc" ? diff : -diff
    })
    return result
  }, [bugs, severityFilter, platformFilter, severitySort])

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-x-3 gap-y-2 items-center rounded-lg border border-border bg-muted/30 px-3 py-2.5">
        <div className="flex items-center gap-1.5 flex-wrap">
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mr-0.5">Severity</span>
          {(["all", "high", "normal", "low", "fixed"] as const).map((s) => (
            <button
              key={s}
              onClick={() => setSeverityFilter(s)}
              className={cn(
                "rounded-full border px-3 py-1 text-xs font-medium transition-colors capitalize",
                severityFilter === s
                  ? s === "all"
                    ? "bg-foreground text-background border-foreground"
                    : SEVERITY_STYLES[s]
                  : s === "all"
                    ? "bg-background text-muted-foreground border-border hover:text-foreground hover:border-foreground/40"
                    : SEVERITY_STYLES_INACTIVE[s],
              )}
            >
              {s === "all" ? "All" : s}
            </button>
          ))}
        </div>
        {PLATFORM_GROUPS.map(({ label, platforms }) => (
          <React.Fragment key={label}>
            <div className="w-px h-4 bg-border" />
            <div className="flex items-center gap-1.5 flex-wrap">
              <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mr-0.5">{label}</span>
              {platforms.map((p) => (
                <button
                  key={p}
                  onClick={() => togglePlatform(p as Platform)}
                  className={cn(
                    "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
                    platformFilter.has(p as Platform)
                      ? cn(PLATFORM_STYLES[p], "border-current/40")
                      : PLATFORM_STYLES_INACTIVE[p],
                  )}
                >
                  {PLATFORM_LABEL[p]}
                </button>
              ))}
            </div>
          </React.Fragment>
        ))}
        <span className="text-xs text-muted-foreground ml-auto">
          {filtered.length} bug{filtered.length !== 1 ? "s" : ""}
        </span>
      </div>

      <div className="rounded-lg border border-border overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-20">
                <button
                  onClick={() => setSeveritySort((d) => d === "asc" ? "desc" : "asc")}
                  className="flex items-center gap-1 hover:text-foreground transition-colors"
                >
                  Severity
                  <span className="text-muted-foreground/60 text-[10px] leading-none">
                    {severitySort === "asc" ? "▲" : "▼"}
                  </span>
                </button>
              </TableHead>
              <TableHead>Bug</TableHead>
              <TableHead className="w-20">Mode</TableHead>
              <TableHead className="w-28">Platform</TableHead>
              <TableHead className="w-14">Version</TableHead>
              <TableHead className="w-20">Date</TableHead>
              <TableHead className="w-36">Media</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} className="py-10 text-center text-muted-foreground whitespace-normal">
                  No bugs match the current filter.
                </TableCell>
              </TableRow>
            )}
            {filtered.map((bug, i) => {
              return (
                <TableRow key={i} className="align-top">
                  <TableCell className="pt-3">
                    {bug.severity && (
                      <span
                        className={cn(
                          "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium capitalize",
                          SEVERITY_STYLES[bug.severity] ?? "bg-muted text-muted-foreground border-border",
                        )}
                      >
                        {bug.severity}
                      </span>
                    )}
                  </TableCell>
                  <TableCell className="whitespace-normal min-w-64 max-w-sm pt-3">
                    <p className="font-medium text-sm text-foreground leading-snug">{bug.title}</p>
                    {bug.details && (
                      <p className="text-xs text-muted-foreground mt-1 leading-relaxed">{bug.details}</p>
                    )}
                    {bug.notes && (
                      <p className="text-xs text-yellow-600 dark:text-yellow-400 mt-1 italic">{bug.notes}</p>
                    )}
                  </TableCell>
                  <TableCell className="pt-3">
                    <div className="flex flex-wrap gap-1">
                      {MODE_PLATFORMS.filter((p) => bug[p]).map((p) => (
                        <span key={p} className={cn("inline-block rounded px-1.5 py-0.5 text-xs font-medium", PLATFORM_STYLES[p])}>
                          {PLATFORM_LABEL[p]}
                        </span>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className="pt-3">
                    <div className="flex flex-wrap gap-1">
                      {DEVICE_PLATFORMS.filter((p) => bug[p]).map((p) => (
                        <span key={p} className={cn("inline-block rounded px-1.5 py-0.5 text-xs font-medium", PLATFORM_STYLES[p])}>
                          {PLATFORM_LABEL[p]}
                        </span>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground pt-3">{bug.version}</TableCell>
                  <TableCell className="text-xs text-muted-foreground pt-3">{bug.date}</TableCell>
                  <TableCell className="pt-3">
                    <div className="flex flex-wrap gap-1.5">
                      {(bug.media ?? []).map((src, mi) => isDiscordLink(src) ? (
                          <a
                            key={mi}
                            href={src}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="relative shrink-0 flex items-center justify-center h-16 w-24 rounded border border-border hover:border-[#5865F2]/60 bg-[#5865F2]/5 hover:bg-[#5865F2]/10 transition-colors"
                            title="View on Discord"
                            data-umami-event="bugs_media_open"
                          >
                            <svg className="size-6 text-[#5865F2]" viewBox="0 0 24 24" fill="currentColor">
                              <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028 14.09 14.09 0 0 0 1.226-1.994.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03z"/>
                            </svg>
                          </a>
                        ) : (
                          <button
                            key={mi}
                            onClick={() => setLightboxSrc(src)}
                            className="relative shrink-0 rounded overflow-hidden border border-border hover:border-primary transition-colors"
                            data-umami-event="bugs_media_open"
                          >
                            {isVideo(src) ? (
                              <>
                                <video
                                  src={src + "#t=0.1"}
                                  className="h-16 w-24 object-cover"
                                  preload="auto"
                                  muted
                                />
                                <span className="absolute inset-0 flex items-center justify-center bg-black/30">
                                  <svg className="size-5 text-white drop-shadow" viewBox="0 0 24 24" fill="currentColor">
                                    <path d="M8 5v14l11-7z"/>
                                  </svg>
                                </span>
                              </>
                            ) : (
                              <img
                                src={src}
                                alt={`screenshot ${mi + 1}`}
                                className="h-16 w-24 object-cover"
                              />
                            )}
                          </button>
                        )
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>

      <Lightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />
    </div>
  )
}
