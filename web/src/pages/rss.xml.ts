import rss from "@astrojs/rss"
import type { APIContext } from "astro"
import { getGuildsSortedByScore, getSolosSortedByScore } from "@/lib/guilds"
import { getBlueprintsSortedByScore } from "@/lib/blueprints"
import { parseLastModified } from "@/lib/dates"
import type { RankedGuild } from "@/types/guild"
import type { RankedBlueprint } from "@/types/blueprint"

const FALLBACK_DATE = new Date("2025-01-01T00:00:00Z")

function safeDate(raw: string | undefined): Date {
  const ms = parseLastModified(raw)
  return ms ? new Date(ms) : FALLBACK_DATE
}

function imageMimeType(url: string): string {
  const ext = url.split(".").pop()?.toLowerCase().split("?")[0]
  if (ext === "png") return "image/png"
  if (ext === "gif") return "image/gif"
  if (ext === "webp") return "image/webp"
  return "image/jpeg"
}

function enclosureFor(image: string | undefined) {
  return image ? { url: image, type: imageMimeType(image), length: 0 } : undefined
}

const DISCORD_INVITE = "https://discord.gg/Qygt9u26Bn"
const MORE_ON_PREFIX = `More on <a href="${DISCORD_INVITE}">WBM Discord</a>:`

function buildItems(entries: RankedGuild[], kind: "guilds" | "solos") {
  return entries.map((g) => {
    const image = g.coverImage ?? g.screenshots?.[0]
    const builders = g.builders?.join(", ") || "Unknown"
    const desc = [
      image ? `<img src="${image}" alt="${g.name}" />` : "",
      `Builder: ${builders}`,
      `${MORE_ON_PREFIX} <a href="${g.discordThread}">${g.discordThread}</a>`,
    ]
      .filter(Boolean)
      .join("<br/>")

    return {
      title: `${g.guildName || g.name} — ${kind === "guilds" ? "Guild Base" : "Solo Build"}`,
      link: `/${kind}/${g.slug}`,
      description: desc,
      pubDate: safeDate(g.lastModified),
      enclosure: enclosureFor(image),
    }
  })
}

function buildBlueprintItems(entries: RankedBlueprint[]) {
  return entries.map((bp) => {
    const image = bp.coverImage ?? bp.screenshots?.[0]
    const desc = [
      image ? `<img src="${image}" alt="${bp.name}" />` : "",
      `Builder: ${bp.builderName || "Unknown"}`,
      `${MORE_ON_PREFIX} <a href="${bp.discordThread}">${bp.discordThread}</a>`,
    ]
      .filter(Boolean)
      .join("<br/>")

    return {
      title: `${bp.name} — Blueprint`,
      link: `/blueprints/${bp.slug}`,
      description: desc,
      pubDate: safeDate(bp.lastModified),
      enclosure: enclosureFor(image),
    }
  })
}

export async function GET(context: APIContext) {
  const guilds = getGuildsSortedByScore()
  const solos = getSolosSortedByScore()
  const blueprints = getBlueprintsSortedByScore()

  // Take the 30 most recently added from each (JSON file order, reversed = newest first)
  const recentGuilds = [...guilds].reverse().slice(0, 30)
  const recentSolos = [...solos].reverse().slice(0, 30)
  const recentBlueprints = [...blueprints].reverse().slice(0, 30)

  const items = [
    ...buildItems(recentGuilds, "guilds"),
    ...buildItems(recentSolos, "solos"),
    ...buildBlueprintItems(recentBlueprints),
  ].sort((a, b) => b.pubDate.getTime() - a.pubDate.getTime()).slice(0, 50)

  return rss({
    title: "Where Builders Meet — New Submissions",
    description: "Latest guild bases, solo builds, and blueprints added to the Where Winds Meet community showcase.",
    site: context.site!,
    items,
    customData: `<language>en-us</language>`,
  })
}
