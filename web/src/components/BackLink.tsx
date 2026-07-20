import * as React from "react"
import { useEffect, useState } from "react"
import { ArrowLeftIcon, ImagesSquareIcon } from "@phosphor-icons/react"
import { url } from "@/lib/url"
import { buttonVariants } from "@/components/ui/button"

const linkClass = buttonVariants({ variant: "outline", size: "sm" })

// An explicit, always-present link straight to the gallery — separate
// from BackLink, which only goes one step back (e.g. builder profile →
// the diagram you came from, not necessarily the gallery itself).
export function GalleryLink() {
  return (
    <a href={url("/gallery")} className={linkClass}>
      <ImagesSquareIcon weight="bold" className="size-3.5" /> Gallery
    </a>
  )
}

// A "back" link that actually goes back, instead of a hardcoded
// destination (the old "Back to gallery" text was wrong as soon as
// navigation went gallery → diagram → builder → another diagram — it
// always claimed "gallery" no matter where you'd actually come from).
//
// Same-origin document.referrer means the visitor navigated here from
// another page on this site, so browser history genuinely has
// somewhere meaningful to go — use that, with a generic "Back" label
// since we don't track exactly what that destination is. No referrer
// (or a foreign one — a shared link opened cold from Discord, a new
// tab, etc.) means there's nothing to go back to, so fall back to a
// real link instead. Heuristic, not a guarantee — some privacy
// settings strip document.referrer even for same-site navigation,
// which just means those visitors see the fallback slightly more
// often than strictly necessary, never anything broken.
export function BackLink({
  fallbackHref = url("/gallery"),
  fallbackLabel = "Back to gallery",
}: {
  fallbackHref?: string
  fallbackLabel?: string
}) {
  const [canGoBack, setCanGoBack] = useState(false)

  useEffect(() => {
    setCanGoBack(document.referrer.startsWith(location.origin))
  }, [])

  if (canGoBack) {
    return (
      <button onClick={() => history.back()} className={`${linkClass} cursor-pointer`}>
        <ArrowLeftIcon weight="bold" className="size-3.5" /> Back
      </button>
    )
  }

  return (
    <a href={fallbackHref} className={linkClass}>
      <ArrowLeftIcon weight="bold" className="size-3.5" /> {fallbackLabel}
    </a>
  )
}
