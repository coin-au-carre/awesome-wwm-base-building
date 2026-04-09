export interface TutorialVideo {
  title: string
  url: string
  description?: string
  tags?: string[]
  author?: string
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
]
