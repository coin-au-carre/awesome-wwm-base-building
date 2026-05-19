import { useRef, useState, useEffect, useCallback, useMemo } from "react"
import { toPng } from "html-to-image"
import QRCodeLib from "qrcode"
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

function renderLore(text: string): string {
  return text
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/~~(.+?)~~/g, "<del>$1</del>")
    .replace(/\*(.+?)\*/g, "<em>$1</em>")
    .replace(/_(.+?)_/g, "<em>$1</em>")
    .replace(/\n\n+/g, "<br><br>")
    .replace(/\n/g, "<br>")
}

// Orange filter matching the community flyer catalog/tutorial icons
const ORANGE_FILTER = "brightness(0) saturate(100%) invert(60%) sepia(80%) saturate(600%) hue-rotate(340deg) drop-shadow(0 0 6px rgba(255,120,20,0.5))"

function FlyerCanvas({
  guild,
  guildUrl,
  displayName,
  buildersStr,
  siteBase,
  qrDataUrl,
  images,
}: Props & { qrDataUrl: string; images: string[] }) {
  const gridCols = images.length <= 2 ? (images.length || 1) : 3
  const gridRows = images.length <= 3 ? 1 : 2
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
        fontFamily: "'Crimson Text', Georgia, serif",
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
          width: 640,
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
          width: 640,
          background: "linear-gradient(to right, rgba(9,9,11,1) 0%, rgba(9,9,11,0.2) 20%, transparent 40%)",
          zIndex: 2,
          pointerEvents: "none",
        }}
      />
      {/* Mosaic top/bottom fade */}
      <div
        style={{
          position: "absolute",
          right: 0, top: 0, bottom: 0,
          width: 640,
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
          width: 620,
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
            src={`${siteBase}/images/logo_3.png`}
            crossOrigin="anonymous"
            alt=""
            style={{
              width: 72,
              height: 72,
              objectFit: "contain",
              flexShrink: 0,
              marginTop: 2,
              filter: "drop-shadow(0 0 18px rgba(236,72,153,0.35))",
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
                <div style={{ fontSize: 15, color: "rgba(255,255,255,0.5)", fontStyle: "italic" }}>
                  by {buildersStr}
                </div>
              )}
              {guild.id && (
                <div
                  style={{
                    fontFamily: "'Cinzel', serif",
                    fontSize: 9,
                    letterSpacing: "1.5px",
                    color: "rgba(255,255,255,0.22)",
                    padding: "2px 7px",
                    border: "1px solid rgba(255,255,255,0.1)",
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
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap", marginBottom: 18 }}>
            {tags.map((tag) => (
              <span
                key={tag}
                style={{
                  padding: "3px 10px",
                  borderRadius: 999,
                  background: "rgba(255,255,255,0.06)",
                  border: "1px solid rgba(255,255,255,0.1)",
                  fontFamily: "'Cinzel', serif",
                  fontSize: 9,
                  letterSpacing: "2px",
                  color: "rgba(255,255,255,0.45)",
                  textTransform: "uppercase",
                }}
              >
                {tag}
              </span>
            ))}
          </div>
        )}

        {/* Lore */}
        {guild.lore && (
          <div
            dangerouslySetInnerHTML={{ __html: renderLore(guild.lore) }}
            style={{
              fontSize: 13.5,
              lineHeight: 1.6,
              color: "rgba(255,255,255,0.42)",
              fontStyle: "italic",
              maxWidth: 460,
              display: "-webkit-box",
              WebkitLineClamp: hasBoth ? 4 : 6,
              WebkitBoxOrient: "vertical",
              overflow: "hidden",
              marginBottom: 12,
            }}
          />
        )}

        {/* What to Visit */}
        {guild.whatToVisit && (
          <div style={{ marginBottom: 16 }}>
            <div
              style={{
                fontFamily: "'Cinzel', serif",
                fontSize: 9,
                letterSpacing: "3px",
                color: "rgba(255,255,255,0.28)",
                textTransform: "uppercase",
                marginBottom: 5,
              }}
            >
              ✦ &nbsp; What to visit
            </div>
            <div
              dangerouslySetInnerHTML={{ __html: renderLore(guild.whatToVisit) }}
              style={{
                fontSize: 13,
                lineHeight: 1.6,
                color: "rgba(255,255,255,0.4)",
                maxWidth: 460,
                display: "-webkit-box",
                WebkitLineClamp: hasBoth ? 10 : 14,
                WebkitBoxOrient: "vertical",
                overflow: "hidden",
              }}
            />
          </div>
        )}

        <div style={{ flex: 1 }} />

        {/* Divider */}
        <div
          style={{
            width: "100%",
            height: 1,
            background: "linear-gradient(to right, rgba(255,255,255,0.12), transparent)",
            marginBottom: 12,
            flexShrink: 0,
          }}
        />

        {/* ── Bottom row: feature grid left, QR code right ── */}
        <div style={{ display: "flex", alignItems: "flex-end", gap: 16 }}>

          {/* Left: label + 2×2 icon-only grid */}
          <div style={{ flex: 1 }}>
            <div
              style={{
                fontFamily: "'Cinzel', serif",
                fontSize: 9,
                letterSpacing: "3px",
                color: "rgba(255,255,255,0.28)",
                textTransform: "uppercase",
                marginBottom: 8,
              }}
            >
              ✦ &nbsp; See more on wherebuildersmeet.com
            </div>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 6 }}>

              {/* Guild Showcases */}
              <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "7px 10px", borderRadius: 8, background: "rgba(255,255,255,0.05)", border: "1px solid rgba(255,255,255,0.08)" }}>
                <img src={`${siteBase}/images/logo_1.png`} crossOrigin="anonymous" alt="" style={{ width: 28, height: 28, objectFit: "contain", flexShrink: 0 }} />
                <div style={{ fontFamily: "'Cinzel', serif", fontSize: 11, fontWeight: 700, color: "rgba(255,255,255,0.75)", letterSpacing: "0.2px" }}>Guilds</div>
              </div>

              {/* Solo Builds */}
              <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "7px 10px", borderRadius: 8, background: "rgba(255,255,255,0.05)", border: "1px solid rgba(255,255,255,0.08)" }}>
                <img src={`${siteBase}/images/logo_mountain1.png`} crossOrigin="anonymous" alt="" style={{ width: 28, height: 28, objectFit: "contain", flexShrink: 0 }} />
                <div style={{ fontFamily: "'Cinzel', serif", fontSize: 11, fontWeight: 700, color: "rgba(255,255,255,0.75)", letterSpacing: "0.2px" }}>Solos</div>
              </div>

              {/* Tutorials */}
              <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "7px 10px", borderRadius: 8, background: "rgba(255,255,255,0.05)", border: "1px solid rgba(255,255,255,0.08)" }}>
                <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 256 256" fill="rgba(255,255,255,0.85)" style={{ filter: ORANGE_FILTER, flexShrink: 0 }}>
                  <path d="M208,24H72A32,32,0,0,0,40,56V224a8,8,0,0,0,8,8H192a8,8,0,0,0,0-16H56a16,16,0,0,1,16-16H208a8,8,0,0,0,8-8V32A8,8,0,0,0,208,24Zm-8,160H72a31.82,31.82,0,0,0-16,4.29V56A16,16,0,0,1,72,40H200Z" opacity={0.3} />
                  <path d="M208,24H72A32,32,0,0,0,40,56V224a8,8,0,0,0,8,8H192a8,8,0,0,0,0-16H56a16,16,0,0,1,16-16H208a8,8,0,0,0,8-8V32A8,8,0,0,0,208,24Zm-8,160H72a31.82,31.82,0,0,0-16,4.29V56A16,16,0,0,1,72,40H200ZM96,88a8,8,0,0,1,8-8h64a8,8,0,0,1,0,16H104A8,8,0,0,1,96,88Zm0,32a8,8,0,0,1,8-8h64a8,8,0,0,1,0,16H104A8,8,0,0,1,96,120Z" />
                </svg>
                <div style={{ fontFamily: "'Cinzel', serif", fontSize: 11, fontWeight: 700, color: "rgba(255,255,255,0.75)", letterSpacing: "0.2px" }}>Tutorials</div>
              </div>

              {/* Construction Catalog */}
              <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "7px 10px", borderRadius: 8, background: "rgba(255,255,255,0.05)", border: "1px solid rgba(255,255,255,0.08)" }}>
                <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 256 256" fill="rgba(255,255,255,0.85)" style={{ filter: ORANGE_FILTER, flexShrink: 0 }}>
                  <path d="M216,48H40a8,8,0,0,0-8,8V200a16,16,0,0,0,16,16H208a16,16,0,0,0,16-16V56A8,8,0,0,0,216,48ZM112,192H48V160h64Zm0-48H48V112h64Zm0-48H48V64h64Zm96,96H128V160h80Zm0-48H128V112h80Zm0-48H128V64h80Z" opacity={0.3} />
                  <path d="M216,40H40A16,16,0,0,0,24,56V200a16,16,0,0,0,16,16H216a16,16,0,0,0,16-16V56A16,16,0,0,0,216,40ZM112,200H48V168h64Zm0-48H48V120h64Zm0-48H48V72h64Zm96,96H128V168h80Zm0-48H128V120h80Zm0-48H128V72h80Z" />
                </svg>
                <div style={{ fontFamily: "'Cinzel', serif", fontSize: 11, fontWeight: 700, color: "rgba(255,255,255,0.75)", letterSpacing: "0.2px" }}>Catalog</div>
              </div>

            </div>
          </div>

          {/* Right: QR code */}
          <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 5, flexShrink: 0 }}>
            <div
              style={{
                fontFamily: "'Cinzel', serif",
                fontSize: 9,
                letterSpacing: "2.5px",
                color: "rgba(255,255,255,0.3)",
                textTransform: "uppercase",
              }}
            >
              View this guild
            </div>
            <div
              style={{
                background: "rgba(9,9,11,0.85)",
                border: "1px solid rgba(255,255,255,0.12)",
                borderRadius: 8,
                padding: 7,
              }}
            >
              {qrDataUrl && (
                <img src={qrDataUrl} alt="QR code" style={{ width: 84, height: 84, display: "block" }} />
              )}
            </div>
            <div
              style={{
                fontFamily: "'Cinzel', serif",
                fontSize: 9,
                letterSpacing: "1.5px",
                color: "rgba(255,255,255,0.35)",
                textAlign: "center",
              }}
            >
              {urlLabel}
            </div>
          </div>
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
    </div>
  )
}

export default function GuildFlyer({ guild, guildUrl, displayName, buildersStr, siteBase }: Props) {
  const [open, setOpen] = useState(false)
  const [qrDataUrl, setQrDataUrl] = useState("")
  const [downloading, setDownloading] = useState(false)
  const flyerRef = useRef<HTMLDivElement>(null)
  const previewContainerRef = useRef<HTMLDivElement>(null)
  const [previewScale, setPreviewScale] = useState(0.5)

  useEffect(() => {
    const el = previewContainerRef.current
    if (!el) { return }
    const ro = new ResizeObserver(([entry]) => {
      setPreviewScale(entry.contentRect.width / W)
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

  useEffect(() => {
    QRCodeLib.toDataURL(guildUrl, {
      width: 168,
      margin: 1,
      color: { dark: "#ffffff", light: "#09090b" },
      errorCorrectionLevel: "M",
    }).then(setQrDataUrl)
  }, [guildUrl])

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

  const flyerProps = { guild, guildUrl, displayName, buildersStr, siteBase, qrDataUrl, images }

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
        Flyer
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

          {/* Hidden full-size for capture */}
          <div style={{ position: "absolute", left: -9999, top: -9999, pointerEvents: "none" }} aria-hidden>
            <div ref={flyerRef}>
              <FlyerCanvas {...flyerProps} />
            </div>
          </div>

          <div className="flex items-center justify-between gap-3 pt-1">
            {allImages.length > 1 && (
              <button
                onClick={shuffle}
                className="inline-flex items-center gap-2 rounded-md border border-white/10 bg-white/5 px-3 py-2 text-sm text-white/60 transition-colors hover:bg-white/10 hover:text-white/80"
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
                disabled={downloading || !qrDataUrl}
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
                    Download PNG
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
