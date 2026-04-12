import rss from "@astrojs/rss"
import type { APIContext } from "astro"
import { getGuildsSortedByScore, getSolosSortedByScore } from "@/lib/guilds"
import type { RankedGuild } from "@/types/guild"

const FALLBACK_DATE = new Date("2025-01-01T00:00:00Z")

function safeDate(raw: string | undefined): Date {
  if (!raw) return FALLBACK_DATE
  const d = new Date(raw)
  return isNaN(d.getTime()) ? FALLBACK_DATE : d
}

function buildItems(entries: RankedGuild[], kind: "guilds" | "solos") {
  return entries.map((g) => {
    const image = g.coverImage ?? g.screenshots?.[0]
    const builders = g.builders?.join(", ") || "Unknown"
    const desc = [
      g.lore ?? `${kind === "guilds" ? "Guild base" : "Solo build"} showcase for ${g.name} in Where Winds Meet.`,
      `Builders: ${builders}`,
      `Score: ${g.score}`,
      g.tags?.length ? `Tags: ${g.tags.join(", ")}` : "",
      image ? `<img src="${image}" alt="${g.name}" />` : "",
    ]
      .filter(Boolean)
      .join("<br/>")

    return {
      title: `${g.guildName || g.name} — ${kind === "guilds" ? "Guild Base" : "Solo Build"}`,
      link: `/${kind}/${g.slug}`,
      description: desc,
      pubDate: safeDate(g.lastModified),
    }
  })
}

export async function GET(context: APIContext) {
  const guilds = getGuildsSortedByScore()
  const solos = getSolosSortedByScore()

  // Take the 30 most recently added from each (JSON file order, reversed = newest first)
  const recentGuilds = [...guilds].reverse().slice(0, 30)
  const recentSolos = [...solos].reverse().slice(0, 30)

  const items = [
    ...buildItems(recentGuilds, "guilds"),
    ...buildItems(recentSolos, "solos"),
  ].sort((a, b) => b.pubDate.getTime() - a.pubDate.getTime()).slice(0, 50)

  return rss({
    title: "Where Builders Meet — New Submissions",
    description: "Latest guild bases and solo builds added to the Where Winds Meet community showcase.",
    site: context.site!,
    items,
    customData: `<language>en-us</language>`,
  })
}
