import * as React from "react"
import { YoutubeLogoIcon, TiktokLogoIcon } from "@phosphor-icons/react"
import type { Channel } from "@/lib/videos"

// A builder's own channel(s) from lib/videos.ts (see lib/builders.ts's
// getBuilderProfile for the matching). Shared between the full builder
// profile page (/builders/[slug], no client directive needed — purely
// presentational, no interactivity) and the builders directory's detail
// panel/sheet (BuildersDirectory.tsx). Deliberately excludes
// tutorialVideos — those are tied to written tutorial articles (e.g.
// carnii, AegisNite), not the builder's own channel, and belong in the
// Tutorials section instead.
export function BuilderMedia({ channels }: { channels: Channel[] }) {
  if (channels.length === 0) { return null }
  return (
    <div className="flex flex-wrap gap-2">
      {channels.map((c) => (
        <a
          key={c.url}
          href={c.url}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1.5 rounded-full ring-1 ring-red-500/25 bg-red-500/5 hover:ring-red-500/50 hover:bg-red-500/10 transition-all px-3 py-1.5 text-xs font-medium"
          data-umami-event="builder_profile_channel_click"
          data-umami-event-name={c.name}
        >
          {c.platform === "tiktok" ? (
            <TiktokLogoIcon weight="fill" className="size-3.5" />
          ) : (
            <YoutubeLogoIcon weight="fill" className="size-3.5 text-red-500" />
          )}
          {c.handle ?? c.name}
        </a>
      ))}
    </div>
  )
}
