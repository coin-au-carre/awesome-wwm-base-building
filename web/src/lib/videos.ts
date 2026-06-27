export interface TutorialVideo {
  title: string
  url: string
  description?: string
  tags?: string[]
  author?: string
  featured?: boolean
}

export interface Channel {
  name: string
  url: string
  handle?: string
  platform?: "youtube" | "tiktok"
  builderSlug?: string
}

export interface TwitchStreamer {
  name: string
  url: string
  login: string
}

export const tutorialVideos: TutorialVideo[] = [
  {
    title: "How to Start Building a House",
    url: "https://www.youtube.com/watch?v=G4vR78IfkfI",
    author: "Azzel83",
    tags: ["beginner"],
  },
  {
    title: "Building a Home by the Waterfall",
    url: "https://www.youtube.com/watch?v=L7oKjiS_vk4",
    author: "Azzel83",
    tags: ["beginner"],
  },
  {
    title: "Build an Underground Passage",
    url: "https://www.tiktok.com/@kaitingliwwm/video/7624968288275156244",
    author: "KaiTingLi",
    tags: ["advanced"],
  },
  {
    title: "WWM CN vs Global Building: You Won't Want to Build on Global Until These Tools Arrive",
    url: "https://youtu.be/HqqSh6cTbdU",
    author: "carnii",
    tags: ["cn", "guild"],
  },
  {
    title: "AegisNite Building System Guide with v1.7 Changes",
    url: "https://www.tiktok.com/@aegisnitewwm/video/7648377225657486624",
    author: "AegisNite",
    tags: ["beginner", "solo", "guild"],
    featured: true,
  },
]

export const channels: Channel[] = [
  {
    name: "AegisNite",
    url: "https://www.tiktok.com/@aegisnitewwm",
    handle: "@aegisnitewwm",
    platform: "tiktok",
    builderSlug: "aegisnite",
  },
  {
    name: "Anarky",
    url: "https://www.tiktok.com/@anarky64",
    handle: "@anarky64",
    platform: "tiktok",
    builderSlug: "anarky",
  },
  {
    name: "Azzel83",
    url: "https://www.youtube.com/channel/UChe01CqFE3129LiUcsX-xQA",
    handle: "@azzel83",
  },
  {
    name: "Chumimaru",
    url: "https://www.tiktok.com/@chumimaru",
    handle: "@chumimaru",
    platform: "tiktok",
    builderSlug: "chumimaru",
  },
  {
    name: "Carnii",
    url: "https://www.youtube.com/@heyu5152",
    handle: "@heyu5152",
    builderSlug: "carnii",
  },
  {
    name: "FoxiKate",
    url: "https://www.youtube.com/@FoxiKate",
    handle: "@FoxiKate",
  },
  {
    name: "imwokay",
    url: "https://www.tiktok.com/@imwokay",
    handle: "@imwokay",
    platform: "tiktok",
    builderSlug: "你wokay吗",
  },
  {
    name: "KaiTingLi",
    url: "https://www.tiktok.com/@kaitingliwwm",
    handle: "@kaitingliwwm",
    platform: "tiktok",
    builderSlug: "kaitingli",
  },
  {
    name: "Marvelite",
    url: "https://www.youtube.com/@Marvelite/videos",
    handle: "@Marvelite",
  },
]

export const twitchStreamers: TwitchStreamer[] = [
  {
    name: "Darth Imperious",
    url: "https://www.twitch.tv/darth_imperious01",
    login: "darth_imperious01",
  },
  {
    name: "mooonlightmage",
    url: "https://www.twitch.tv/mooonlightmage",
    login: "mooonlightmage",
  },
]
