import * as React from "react"
import { useState, useEffect } from "react"
import Autoplay from "embla-carousel-autoplay"
import { StackIcon } from "@phosphor-icons/react"
import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselPrevious,
  CarouselNext,
} from "@/components/ui/carousel"
import { Skeleton } from "@/components/ui/skeleton"
import { formatBuilderName, stripGuildShowcase, thumbUrl } from "@/lib/format"

export interface CarouselGuild {
  slug: string
  rank: number
  name: string
  guildName?: string
  builders: string[]
  tags?: string[]
  coverImage?: string
  screenshots?: string[]
  buildTitle?: string
  createdAt?: string
  pricingLabel?: string
}

import { url } from "@/lib/url"

const TAG_PALETTE = [
  { bg: "bg-violet-500/20", text: "text-violet-200", ring: "ring-1 ring-inset ring-violet-400/40" },
  { bg: "bg-sky-500/20", text: "text-sky-200", ring: "ring-1 ring-inset ring-sky-400/40" },
  { bg: "bg-emerald-500/20", text: "text-emerald-200", ring: "ring-1 ring-inset ring-emerald-400/40" },
  { bg: "bg-amber-500/20", text: "text-amber-200", ring: "ring-1 ring-inset ring-amber-400/40" },
  { bg: "bg-rose-500/20", text: "text-rose-200", ring: "ring-1 ring-inset ring-rose-400/40" },
  { bg: "bg-teal-500/20", text: "text-teal-200", ring: "ring-1 ring-inset ring-teal-400/40" },
  { bg: "bg-orange-500/20", text: "text-orange-200", ring: "ring-1 ring-inset ring-orange-400/40" },
  { bg: "bg-indigo-500/20", text: "text-indigo-200", ring: "ring-1 ring-inset ring-indigo-400/40" },
]
function tagColor(tag: string) {
  const hash = [...tag].reduce((a, c) => a + c.charCodeAt(0), 0)
  return TAG_PALETTE[hash % TAG_PALETTE.length]
}

function relativeDate(createdAt: string): string {
  const ms = Date.now() - new Date(createdAt.replace(" at ", " ")).getTime()
  if (isNaN(ms) || ms < 0) return ""
  const hours = Math.floor(ms / 3600000)
  const days = Math.floor(ms / 86400000)
  if (hours < 24) return hours <= 1 ? "1h ago" : `${hours}h ago`
  if (days === 1) return "yesterday"
  if (days < 7) return `${days}d ago`
  if (days < 30) return `${Math.floor(days / 7)}w ago`
  return `${Math.floor(days / 30)}mo ago`
}

interface Props {
  guilds: CarouselGuild[]
  basePath?: string
  showDate?: boolean
  showRank?: boolean
  nameScope?: string
}

export function GuildCarousel({ guilds, basePath = "guilds", showDate = false, showRank = true, nameScope }: Props) {
  const plugin = React.useRef(
    Autoplay({ delay: 3500, stopOnInteraction: true })
  )
  const [mounted, setMounted] = useState(false)
  useEffect(() => setMounted(true), [])

  if (!mounted) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {guilds.slice(0, 3).map((g, idx) => {
          const shots = g.screenshots ?? []
          const img = g.coverImage ?? shots[0]
          return (
            <a
              key={g.slug}
              href={url(`/${basePath}/${g.slug}`)}
              className="group relative block overflow-hidden rounded-xl ring-1 ring-border aspect-video bg-muted"
            >
              {img && (
                <img
                  src={thumbUrl(img, 640, 360)}
                  alt={stripGuildShowcase(g.guildName || g.name)}
                  loading={idx === 0 ? "eager" : "lazy"}
                  className="absolute inset-0 h-full w-full object-cover"
                  style={nameScope ? { viewTransitionName: `${nameScope}-${g.slug}` } : undefined}
                />
              )}
              <div className="absolute inset-0 bg-linear-to-t from-black/70 to-transparent" />
              <div className="absolute bottom-0 left-0 right-0 p-3">
                <p className="text-white font-medium text-sm leading-tight">{stripGuildShowcase(g.guildName || g.name)}</p>
              </div>
            </a>
          )
        })}
      </div>
    )
  }

  return (
    <Carousel
      opts={{ align: "start", loop: true }}
      plugins={[plugin.current]}
      className="w-full"
    >
      <CarouselContent>
        {guilds.map((g, idx) => {
          const shots = g.screenshots ?? []
          const img = g.coverImage ?? shots[0]
          return (
            <CarouselItem key={g.slug} className="basis-full sm:basis-1/2 lg:basis-1/3">
              <a
                href={url(`/${basePath}/${g.slug}`)}
                className="card-glow carousel-fade-up group relative block overflow-hidden rounded-xl ring-1 ring-border aspect-video bg-muted transition-all"
                style={{ animationDelay: `${idx * 40}ms` }}
                onClick={() => window.umami?.track("guild_click", { name: g.name, rank: g.rank, source: "carousel", type: basePath })}
              >
                {img && (
                  <img
                    src={thumbUrl(img, 640, 360)}
                    alt={stripGuildShowcase(g.guildName || g.name)}
                    loading={idx === 0 ? "eager" : "lazy"}
                    fetchPriority={idx === 0 ? "high" : "auto"}
                    decoding="async"
                    onError={(e) => ((e.target as HTMLImageElement).style.display = "none")}
                    className="absolute inset-0 h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                    style={nameScope ? { viewTransitionName: `${nameScope}-${g.slug}` } : undefined}
                  />
                )}
                <div className="absolute inset-0 bg-linear-to-t from-black/70 to-transparent" />
                {showDate && g.createdAt && (
                  <div className="absolute top-2 left-2 z-20 text-[10px] font-medium px-1.5 py-0.5 rounded-md bg-black/50 backdrop-blur-sm text-white/60">
                    {relativeDate(g.createdAt)}
                  </div>
                )}
                <div className="absolute top-2 right-2 z-20 flex items-center gap-1.5">
                  {g.pricingLabel && (() => {
                    const cfg = tagColor(g.pricingLabel)
                    return (
                      <span className={`inline-flex items-center rounded-full ${cfg.bg} ${cfg.text} ${cfg.ring} backdrop-blur-sm px-2 py-0.5 text-[10px] font-medium`}>
                        {g.pricingLabel}
                      </span>
                    )
                  })()}
                  {g.buildTitle && (
                    <div className="flex items-center gap-1 text-[10px] font-medium px-1.5 py-0.5 rounded-md bg-black/50 backdrop-blur-sm text-blue-300">
                      <StackIcon weight="fill" className="size-2.5 shrink-0" />
                      {g.buildTitle}
                    </div>
                  )}
                  {showRank && g.rank <= 10 && (
                    <div className="text-[11px] font-bold font-mono px-1.5 py-0.5 rounded-md bg-black/50 backdrop-blur-sm text-white/90">
                      #{g.rank}
                    </div>
                  )}
                </div>
                <div className="absolute bottom-0 left-0 right-0 p-3">
                  <div className="flex items-baseline gap-1.5 mb-1">
                    <p className="text-white font-medium text-sm leading-tight">{stripGuildShowcase(g.guildName || g.name)}</p>
                    {g.builders && g.builders.length > 0 && (
                      <p className="text-white/60 text-[11px] leading-tight truncate">by {g.builders.map(formatBuilderName).filter(Boolean).join(", ")}</p>
                    )}
                  </div>
                  {g.tags && g.tags.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {g.tags.slice(0, 3).map((tag) => {
                        const cfg = tagColor(tag)
                        return (
                          <span
                            key={tag}
                            className={`inline-flex items-center rounded-full ${cfg.bg} ${cfg.text} ${cfg.ring} backdrop-blur-sm px-2 py-0.5 text-[10px] font-medium`}
                          >
                            {tag}
                          </span>
                        )
                      })}
                    </div>
                  )}
                </div>
              </a>
            </CarouselItem>
          )
        })}
      </CarouselContent>
      <CarouselPrevious className="size-12 -left-5 sm:-left-6" />
      <CarouselNext className="size-12 -right-5 sm:-right-6" />
    </Carousel>
  )
}
