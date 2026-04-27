export const MOD_IDS = new Set([
  "149790526076354561", // Ahlyam
  "721510597958828183", // WindXP
  "376950312721711118", // Babe
])

// Guilds posted by mods that are nonetheless community entries.
export const MOD_EXCEPTIONS = new Set(["Jenova", "PleasureSeeker", "WINDXP Bridge", "Lucid Echoes"])

export function isCommunityPosted(g: { postedOnBehalfOf?: string; posterDiscordId?: string; name: string; guildName?: string }): boolean {
  if (g.postedOnBehalfOf) { return true }
  if (!g.posterDiscordId) { return false }
  if (MOD_EXCEPTIONS.has(g.name) || MOD_EXCEPTIONS.has(g.guildName ?? "")) { return true }
  return !MOD_IDS.has(g.posterDiscordId)
}
