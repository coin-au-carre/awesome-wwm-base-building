export function thumbUrl(src: string, width = 640, height = 360): string {
  try {
    const u = new URL(src)
    if (u.hostname === "cdn.discordapp.com" || u.hostname === "media.discordapp.net") {
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

export function stripGuildShowcase(name: string): string {
  return name.replace(/\bGuild Showcase\b/gi, "").trim()
}

/** Strip Discord mention format: `<@773253240498159636> (GaoQingYang)` → `GaoQingYang`. */
export function formatBuilderName(raw: string): string {
  const match = raw.match(/<@\d+>\s*\((.+?)\)/)
  if (match) { return match[1].trim() }
  if (/^<@\d+>$/.test(raw.trim())) { return "" }
  return raw.trim()
}
