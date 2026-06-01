import { useRef, useState, useEffect, useCallback, useMemo } from "react"
import { toPng } from "html-to-image"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog"
import type { RankedGuild } from "@/types/guild"

interface Props {
  guild: RankedGuild
  guildUrl: string
  displayName: string
  buildersStr: string
  siteBase: string
}

const W = 1200
const H = 670

function shuffleArray<T>(arr: T[]): T[] {
  const a = [...arr]
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]]
  }
  return a
}

function extractFlyerLore(text: string): string {
  // Prefer the blockquote section (the story) over preamble paragraphs.
  const lines = text.split("\n")
  const firstQuoteLine = lines.findIndex((l) => l.trimStart().startsWith(">"))
  if (firstQuoteLine > 0) {
    return lines.slice(firstQuoteLine).join("\n")
  }
  return text
}

function renderLore(text: string): string {
  return text
    .replace(/^>\s?/gm, "")          // strip blockquote markers
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/~~(.+?)~~/g, "<del>$1</del>")
    .replace(/\*(.+?)\*/g, "<em>$1</em>")
    .replace(/_(.+?)_/g, "<em>$1</em>")
    .replace(/\n\n+/g, "<br><br>")
    .replace(/\n/g, "<br>")
}


function FlyerCanvas({
  guild,
  guildUrl,
  displayName,
  buildersStr,
  siteBase,
  images,
}: Props & { images: string[] }) {
  const gridCols = images.length <= 2 ? (images.length || 1) : 3
  const gridRows = images.length <= 3 ? 1 : 2
  const loreFontSize = guild.lore
    ? guild.lore.length > 800 ? 12 : guild.lore.length > 500 ? 13 : 15
    : 15
  const tags = (guild.tags ?? []).slice(0, 4)
  const urlLabel = guildUrl.replace("https://www.", "").replace(/\/$/, "")
  const hasBoth = !!(guild.lore && guild.whatToVisit)

  return (
    <div
      style={{
        width: W,
        height: H,
        position: "relative",
        overflow: "hidden",
        fontFamily: "'Crimson Text', Georgia, serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Noto Color Emoji', sans-serif",
        background: "#09090b",
        flexShrink: 0,
      }}
    >
      {/* Right: screenshot mosaic */}
      <div
        style={{
          position: "absolute",
          right: 0,
          top: 0,
          bottom: 0,
          width: 720,
          display: "grid",
          gridTemplateColumns: `repeat(${gridCols}, 1fr)`,
          gridTemplateRows: `repeat(${gridRows}, 1fr)`,
          gap: 3,
        }}
      >
        {images.map((src, i) => (
          <div key={i} style={{ overflow: "hidden" }}>
            <img
              src={src}
              crossOrigin="anonymous"
              style={{ width: "100%", height: "100%", objectFit: "cover", filter: "brightness(0.85) saturate(0.8)" }}
              alt=""
            />
          </div>
        ))}
        {images.length === 0 && (
          <div style={{ background: "rgba(255,255,255,0.03)", gridColumn: "1/-1", gridRow: "1/-1" }} />
        )}
      </div>

      {/* Mosaic left fade */}
      <div
        style={{
          position: "absolute",
          right: 0, top: 0, bottom: 0,
          width: 720,
          background: "linear-gradient(to right, rgba(9,9,11,1) 0%, rgba(9,9,11,0.2) 22%, transparent 44%)",
          zIndex: 2,
          pointerEvents: "none",
        }}
      />
      {/* Mosaic top/bottom fade */}
      <div
        style={{
          position: "absolute",
          right: 0, top: 0, bottom: 0,
          width: 720,
          background: "linear-gradient(to bottom, rgba(9,9,11,0.5) 0%, transparent 18%), linear-gradient(to top, rgba(9,9,11,0.5) 0%, transparent 18%)",
          zIndex: 2,
          pointerEvents: "none",
        }}
      />

      {/* Left content panel */}
      <div
        style={{
          position: "absolute",
          left: 0, top: 0, bottom: 0,
          width: 680,
          zIndex: 10,
          display: "flex",
          flexDirection: "column",
          justifyContent: "flex-start",
          padding: "36px 56px 52px",
        }}
      >
        {/* ── Top: logo + guild name + builder ── */}
        <div style={{ display: "flex", alignItems: "flex-start", gap: 18, marginBottom: 16 }}>
          <img
            src={`${siteBase}/images/logo_3.webp`}
            crossOrigin="anonymous"
            alt=""
            style={{
              width: 96,
              height: 96,
              objectFit: "contain",
              flexShrink: 0,
              marginTop: 2,
              filter: "drop-shadow(0 0 22px rgba(236,72,153,0.4))",
            }}
          />
          <div>
            <div
              style={{
                fontFamily: "'Cinzel', serif",
                fontWeight: 900,
                fontSize: displayName.length > 22 ? 32 : displayName.length > 16 ? 38 : 46,
                lineHeight: 1.05,
                color: "#ffffff",
                letterSpacing: "1.5px",
                textShadow: "0 2px 24px rgba(0,0,0,0.9)",
                marginBottom: 6,
              }}
            >
              {displayName}
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
              {buildersStr && (
                <div style={{ fontSize: 16, color: "rgba(255,255,255,0.6)", fontStyle: "italic" }}>
                  by {buildersStr}
                </div>
              )}
              {guild.id && (
                <div
                  style={{
                    fontFamily: "'Cinzel', serif",
                    fontSize: 9,
                    letterSpacing: "1.5px",
                    color: "rgba(255,255,255,0.45)",
                    padding: "2px 7px",
                    border: "1px solid rgba(255,255,255,0.2)",
                    borderRadius: 4,
                  }}
                >
                  ID {guild.id}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Tags */}
        {tags.length > 0 && (
          <div style={{ display: "flex", alignItems: "center", gap: 0, marginBottom: 16, marginTop: 4 }}>
            {tags.map((tag, i) => (
              <span key={tag} style={{ display: "flex", alignItems: "center" }}>
                {i > 0 && (
                  <span style={{ color: "rgba(255,255,255,0.18)", margin: "0 8px", fontSize: 10 }}>·</span>
                )}
                <span
                  style={{
                    fontFamily: "'Cinzel', serif",
                    fontSize: 9,
                    letterSpacing: "2.5px",
                    color: "rgba(255,255,255,0.38)",
                    textTransform: "uppercase",
                  }}
                >
                  {tag}
                </span>
              </span>
            ))}
          </div>
        )}

        {/* Lore */}
        {guild.lore && (
          <div
            dangerouslySetInnerHTML={{ __html: renderLore(extractFlyerLore(guild.lore)) }}
            style={{
              fontSize: loreFontSize,
              lineHeight: 1.65,
              color: "rgba(255,255,255,0.5)",
              fontStyle: "italic",
              fontFamily: "'Crimson Text', Georgia, serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Noto Color Emoji', sans-serif",
              maxWidth: 480,
              display: "-webkit-box",
              WebkitLineClamp: hasBoth
                ? (loreFontSize <= 12 ? 9 : loreFontSize <= 13 ? 7 : 6)
                : (loreFontSize <= 12 ? 22 : loreFontSize <= 13 ? 19 : 16),
              WebkitBoxOrient: "vertical",
              overflow: "hidden",
              marginTop: 20,
              marginBottom: 20,
            }}
          />
        )}

        {/* What to Visit */}
        {guild.whatToVisit && (
          <div style={{ marginBottom: 16, display: "flex", gap: 12 }}>
            <div style={{ width: 2, borderRadius: 2, background: "rgba(236,72,153,0.25)", flexShrink: 0, alignSelf: "stretch" }} />
            <div>
              <div
                style={{
                  fontFamily: "'Cinzel', serif",
                  fontSize: 9,
                  letterSpacing: "3px",
                  color: "rgba(255,255,255,0.32)",
                  textTransform: "uppercase",
                  marginBottom: 6,
                }}
              >
                What to visit
              </div>
              <div
                dangerouslySetInnerHTML={{ __html: renderLore(guild.whatToVisit) }}
                style={{
                  fontSize: 14,
                  lineHeight: 1.65,
                  color: "rgba(255,255,255,0.45)",
                  maxWidth: 480,
                  display: "-webkit-box",
                  WebkitLineClamp: hasBoth ? 10 : 14,
                  WebkitBoxOrient: "vertical",
                  overflow: "hidden",
                }}
              />
            </div>
          </div>
        )}

        <div style={{ flex: 1 }} />

        {/* ── Bottom: URL ── */}
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <div
            style={{
              fontFamily: "'Cinzel', serif",
              fontSize: 10,
              letterSpacing: "1px",
              color: "rgba(255,255,255,0.5)",
              textTransform: "lowercase",
              whiteSpace: "nowrap",
            }}
          >
            {urlLabel}
          </div>
          <div style={{ flex: 1, height: 1, background: "linear-gradient(to right, rgba(255,255,255,0.08), transparent)" }} />
        </div>
      </div>

      {/* Corner decorations */}
      {(["tl", "tr", "bl", "br"] as const).map((c) => (
        <div
          key={c}
          style={{
            position: "absolute",
            width: 18,
            height: 18,
            borderColor: "rgba(255,255,255,0.15)",
            borderStyle: "solid",
            zIndex: 20,
            ...(c === "tl" ? { top: 26, left: 26, borderWidth: "1.5px 0 0 1.5px" } : {}),
            ...(c === "tr" ? { top: 26, right: 26, borderWidth: "1.5px 1.5px 0 0" } : {}),
            ...(c === "bl" ? { bottom: 26, left: 26, borderWidth: "0 0 1.5px 1.5px" } : {}),
            ...(c === "br" ? { bottom: 26, right: 26, borderWidth: "0 1.5px 1.5px 0" } : {}),
          }}
        />
      ))}

      {/* Branding — vertical right edge */}
      <div
        style={{
          position: "absolute",
          right: 3,
          top: "50%",
          transform: "translateY(-50%)",
          writingMode: "vertical-rl",
          zIndex: 21,
          fontFamily: "'Cinzel', serif",
          fontSize: 8,
          letterSpacing: "3px",
          color: "rgba(255,255,255,0.28)",
          textTransform: "lowercase",
          whiteSpace: "nowrap",
          userSelect: "none",
        }}
      >
        wherebuildersmeet.com
      </div>
    </div>
  )
}

export default function GuildFlyer({ guild, guildUrl, displayName, buildersStr, siteBase }: Props) {
  const [open, setOpen] = useState(false)
  const [downloading, setDownloading] = useState(false)
  const flyerRef = useRef<HTMLDivElement>(null)
  const previewContainerRef = useRef<HTMLDivElement>(null)
  const [previewScale, setPreviewScale] = useState(0.5)

  useEffect(() => {
    if (!open) return
    const el = previewContainerRef.current
    if (!el) { return }
    const ro = new ResizeObserver(([entry]) => {
      if (entry.contentRect.width > 0) {
        setPreviewScale(entry.contentRect.width / W)
      }
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [open])

  const allImages = useMemo(() => [
    ...(guild.coverImage ? [guild.coverImage] : []),
    ...(guild.screenshots ?? []).filter((s) => s !== guild.coverImage),
  ], [guild])

  const [images, setImages] = useState<string[]>(() => allImages.slice(0, 6))

  const shuffle = useCallback(() => {
    setImages(shuffleArray(allImages).slice(0, 6))
  }, [allImages])

  const handleDownload = useCallback(async () => {
    if (!flyerRef.current) { return }
    setDownloading(true)
    try {
      const dataUrl = await toPng(flyerRef.current, {
        pixelRatio: 2,
        cacheBust: true,
        fetchRequestInit: { mode: "cors" },
      })
      const a = document.createElement("a")
      a.download = `${guild.slug}-flyer.png`
      a.href = dataUrl
      a.click()
    } finally {
      setDownloading(false)
    }
  }, [guild.slug])

  const flyerProps = { guild, guildUrl, displayName, buildersStr, siteBase, images }

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        title="Download share flyer"
        className="inline-flex items-center gap-1.5 text-white/80 hover:text-white transition-colors px-2.5 py-2 rounded-lg bg-pink-500/20 hover:bg-pink-500/30 border border-pink-500/30 backdrop-blur-sm cursor-pointer shrink-0 text-[13px] font-medium"
        data-umami-event="guild_flyer_open"
        data-umami-event-guild={guild.slug}
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <rect width="18" height="18" x="3" y="3" rx="2" ry="2"/>
          <circle cx="9" cy="9" r="2"/>
          <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/>
        </svg>
      </button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="p-6" style={{ maxWidth: "min(760px, calc(100vw - 2rem))" }}>
          <DialogHeader>
            <DialogTitle className="font-heading text-lg flex items-center gap-2">
              Guild flyer
              <span className="text-[10px] font-semibold uppercase tracking-widest text-amber-500/80 border border-amber-500/30 bg-amber-500/10 rounded px-1.5 py-0.5">Beta</span>
            </DialogTitle>
            <DialogDescription className="text-sm text-muted-foreground">
              Download as PNG to share on Discord or social media.
            </DialogDescription>
          </DialogHeader>

          <div
            ref={previewContainerRef}
            style={{
              width: "100%",
              height: H * previewScale,
              overflow: "hidden",
              borderRadius: 8,
              border: "1px solid rgba(255,255,255,0.08)",
            }}
          >
            <div style={{ transform: `scale(${previewScale})`, transformOrigin: "top left", width: W, height: H }}>
              <FlyerCanvas {...flyerProps} />
            </div>
          </div>

          {/* Hidden full-size for capture — fixed so it's outside the dialog's layout context */}
          <div style={{ position: "fixed", left: -9999, top: -9999, pointerEvents: "none" }} aria-hidden>
            <div ref={flyerRef}>
              <FlyerCanvas {...flyerProps} />
            </div>
          </div>

          <div className="flex items-center justify-between gap-3 pt-1">
            {allImages.length > 1 && (
              <button
                onClick={shuffle}
                className="inline-flex items-center gap-2 rounded-md border border-white/10 bg-white/5 px-3 py-2 text-sm text-white/60 transition-colors hover:bg-white/10 hover:text-white/80"
                data-umami-event="guild_flyer_shuffle"
                data-umami-event-guild={guild.slug}
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M2 18h1.4c1.3 0 2.5-.6 3.3-1.7l6.1-8.6c.7-1.1 2-1.7 3.3-1.7H22"/>
                  <path d="m18 2 4 4-4 4"/>
                  <path d="M2 6h1.9c1.5 0 2.9.9 3.6 2.2"/>
                  <path d="M22 18h-5.9c-1.3 0-2.5-.7-3.1-1.8l-.8-1.4"/>
                  <path d="m18 14 4 4-4 4"/>
                </svg>
                Shuffle
              </button>
            )}

            <div className="flex items-center gap-3 ml-auto">
              <button
                onClick={() => setOpen(false)}
                className="inline-flex items-center rounded-md border border-white/10 bg-white/5 px-4 py-2 text-sm text-white/60 transition-colors hover:bg-white/10 hover:text-white/80"
              >
                Close
              </button>
              <button
                onClick={handleDownload}
                disabled={downloading}
                className="inline-flex items-center gap-2 rounded-md bg-white px-4 py-2 text-sm font-semibold text-black transition-opacity hover:opacity-90 disabled:opacity-50"
                data-umami-event="guild_flyer_download"
                data-umami-event-guild={guild.slug}
              >
                {downloading ? (
                  <>
                    <svg className="animate-spin" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
                    Generating…
                  </>
                ) : (
                  <>
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                    Download
                  </>
                )}
              </button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
