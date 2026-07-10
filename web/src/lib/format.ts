export function thumbUrl(src: string, width = 640, height = 360): string {
  try {
    const u = new URL(src)
    if (u.hostname === "cdn.discordapp.com" || u.hostname === "media.discordapp.net") {
      // media.discordapp.net is Discord's media proxy — it actually resizes and converts.
      // cdn.discordapp.com is a dumb file server that ignores width/height params.
      u.hostname = "media.discordapp.net"
      u.searchParams.set("format", "webp")
      u.searchParams.set("width", String(width))
      u.searchParams.set("height", String(height))
      return u.toString()
    }
  } catch {}
  return src
}

/**
 * Port of Go's Slugify() from internal/discord/spotlight.go.
 * Must produce identical output for all guild names.
 */
export function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, "-")
    .replace(/^-+|-+$/g, "")
}

/** YouTube thumbnail URL for a watch/share link, used as a cover-image fallback when no screenshots exist. */
export function youtubeThumbnail(rawURL: string): string | undefined {
  try {
    const u = new URL(rawURL)
    let id = ""
    if (u.hostname === "youtu.be") { id = u.pathname.replace(/^\//, "") }
    else if (u.hostname === "www.youtube.com" || u.hostname === "youtube.com") { id = u.searchParams.get("v") ?? "" }
    return id ? `https://img.youtube.com/vi/${id}/hqdefault.jpg` : undefined
  } catch {
    return undefined
  }
}

export function stripGuildShowcase(name: string): string {
  return name.replace(/\bGuild Showcase\b/gi, "").trim()
}

/** Strip Discord mention format: `<@773253240498159636> (GaoQingYang)` → `GaoQingYang`. */
export function formatBuilderName(raw: string): string {
  const match = raw.match(/<@\d+>\s*\((.+?)\)/)
  if (match) { return match[1].trim() }
  if (/^<@\d+>$/.test(raw.trim())) { return "" }
  return raw.trim().replace(/\s*\(.*$/, "").trim()
}

/** URL slug for a builder profile page. */
export function builderSlug(name: string): string {
  return slugify(formatBuilderName(name))
}
