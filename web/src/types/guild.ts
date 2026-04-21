export interface ScreenshotSection {
  label?: string
  screenshots: string[]
}

export interface Guild {
  id?: string
  name: string
  guildName?: string
  builders: string[]
  tags?: string[]
  discordThread: string
  builderDiscordId?: string
  posterUsername?: string
  postedOnBehalfOf?: string
  lore?: string
  whatToVisit?: string
  score: number
  reactions?: Record<string, string[]>
  coverImage?: string
  screenshots?: string[]
  screenshotSections?: ScreenshotSection[]
  videos?: string[]
  lastModified?: string
}

export interface RankedGuild extends Guild {
  slug: string
  rank: number
}
