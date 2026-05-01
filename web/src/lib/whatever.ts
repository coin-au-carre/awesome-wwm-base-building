import { readFileSync } from "fs"
import { resolve } from "path"
import { getUsers, getBuilderSearchPath } from "@/lib/guilds"

export interface ReactionDetail {
  emoji: string
  count: number
}

export interface WhateverPost {
  id: string
  authorName: string
  authorId: string
  images: string[]
  reactions: number
  reactionDetails: ReactionDetail[]
  messageUrl: string
  postedAt: string
}

export interface WhateverItem extends WhateverPost {
  displayName: string
  builderPath: string | null
}

function loadPosts(): WhateverPost[] {
  try {
    const raw = readFileSync(resolve(process.cwd(), "..", "data/whatever.json"), "utf-8")
    return JSON.parse(raw)
  } catch {
    return []
  }
}

function resolveDisplayName(post: WhateverPost): string {
  const users = getUsers()
  const user = users[post.authorId]
  if (!user) { return post.authorName }
  return user.nickname ?? user.globalName ?? user.username ?? post.authorName
}

const ALL_POSTS: WhateverPost[] = loadPosts()

export function getWhateverPosts(): WhateverItem[] {
  return [...ALL_POSTS]
    .sort((a, b) => b.reactions - a.reactions)
    .map((p) => {
      const displayName = resolveDisplayName(p)
      return { ...p, displayName, builderPath: getBuilderSearchPath(displayName) }
    })
}

export function getWhateverStats(): { totalImages: number; totalContributors: number } {
  const images = ALL_POSTS.reduce((n, p) => n + p.images.length, 0)
  const contributors = new Set(ALL_POSTS.map((p) => p.authorId)).size
  return { totalImages: images, totalContributors: contributors }
}
