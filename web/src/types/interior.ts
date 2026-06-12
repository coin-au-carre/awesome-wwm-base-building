export interface Interior {
  name: string
  builderName?: string
  builderId?: string
  description?: string
  screenshots?: string[]
  videos?: string[]
  discordThread: string
  createdAt?: string
  lastModified?: string
}

export interface IndexedInterior extends Interior {
  slug: string
}
