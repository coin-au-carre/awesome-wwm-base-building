import * as React from "react"
import { useEffect, useState } from "react"
import { Avatar } from "@/components/GalleryGrid"
import { WBM_RELAY_URL, designerUrl } from "@/lib/gallery"

// Swaps in the builder's real NetEase avatar once fetched, keeping the
// initials fallback until then — only rendered when this builder has a
// linked neteaseNumberId (see [slug].astro).
export function BuilderAvatar({ numberId, initial, className }: { numberId: string; initial: string; className: string }) {
  const [avatarUrl, setAvatarUrl] = useState<string | undefined>(undefined)

  useEffect(() => {
    if (!WBM_RELAY_URL) return
    fetch(designerUrl(numberId))
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => data?.avatar_url && setAvatarUrl(data.avatar_url))
      .catch(() => {})
  }, [numberId])

  if (avatarUrl) return <Avatar src={avatarUrl} className={className} />

  return (
    <div className={`${className} rounded-full bg-primary/10 text-primary items-center justify-center font-bold shrink-0 ring-2 ring-primary/20`}>
      {initial}
    </div>
  )
}
