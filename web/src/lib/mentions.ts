import { builderSlug } from "@/lib/format"
import type { UserMap } from "@/lib/guilds"
import { getBuilderProfile } from "@/lib/builders"
import { resolveCanonical } from "@/lib/builder-aliases"

/** Replace `<@discordId>` mentions with a display name, linked to the builder profile if one exists. */
export function resolveMentions(text: string, users: UserMap, base: string): string {
  return text.replace(/<@(\d+)>/g, (_, id) => {
    const user = users[id]
    if (!user) return `@${id}`
    const name = user.nickname ?? user.globalName ?? user.username
    const canonical = resolveCanonical(builderSlug(name))
    const profile = getBuilderProfile(canonical)
    return profile
      ? `<a href="${base}/builders/${canonical}" class="underline hover:opacity-75 transition-opacity">${name}</a>`
      : name
  })
}
