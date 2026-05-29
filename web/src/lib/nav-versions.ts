import { readFileSync } from "fs"
import { resolve } from "path"

function repoFile(p: string) {
  return resolve(process.cwd(), "..", p)
}

interface NavVersions {
  guilds: string
  solo: string
  moments: string
  blueprints: string
  catalog: string
  tutorials: string
  media: string
  "how-it-works": string
  contribute: string
  ruby: string
  credits: string
}

const FALLBACK: NavVersions = {
  guilds: "0", solo: "0", moments: "0", blueprints: "0", catalog: "0",
  tutorials: "0", media: "0", "how-it-works": "0", contribute: "0",
  ruby: "0", credits: "0",
}

export function getNavVersions(): NavVersions {
  try {
    return JSON.parse(readFileSync(repoFile("data/nav-versions.json"), "utf-8"))
  } catch {
    return FALLBACK
  }
}
