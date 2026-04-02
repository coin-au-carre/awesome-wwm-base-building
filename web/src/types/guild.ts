export interface Guild {
  id?: string
  name: string
  guildName?: string
  builders: string[]
  tags?: string[]
  discordThread: string
  builderDiscordId?: string
  lore?: string
  whatToVisit?: string
  score: number
  screenshots?: string[]
  videos?: string[]
}

export interface RankedGuild extends Guild {
  slug: string
  rank: number
}
