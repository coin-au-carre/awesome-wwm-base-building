export interface ScreenshotSection {
  label?: string
  screenshots: string[]
}

export interface Blueprint {
  name: string
  builderName?: string
  builderId?: string
  price?: string
  isFree?: boolean
  isPayToBuild?: boolean
  materials?: string
  description?: string
  tags?: string[]
  score: number
  coverImage?: string
  screenshots?: string[]
  screenshotSections?: ScreenshotSection[]
  videos?: string[]
  shareCodes?: string[]
  discordThread: string
  createdAt?: string
  lastModified?: string
}

export interface RankedBlueprint extends Blueprint {
  slug: string
  rank: number
}
