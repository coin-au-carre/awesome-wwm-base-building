import { readFileSync } from "fs"
import { getUsers, getBuilderSearchPath } from "@/lib/guilds"

export interface WhateverPost {
  id: string
  authorName: string
  authorId: string
  images: string[]
  reactions: number
  messageUrl: string
  postedAt: string
}

export interface WhateverItem extends WhateverPost {
  displayName: string
  builderPath: string | null
}

function loadPosts(): WhateverPost[] {
  try {
    const raw = readFileSync(new URL("../../../data/whatever.json", import.meta.url), "utf-8")
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
