import * as React from "react"
import Autoplay from "embla-carousel-autoplay"
import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselPrevious,
  CarouselNext,
} from "@/components/ui/carousel"
import type { RankedGuild } from "@/types/guild"
import { formatBuilderName, stripGuildShowcase } from "@/lib/slugify"
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

interface Props {
  guilds: RankedGuild[]
  basePath?: string
}

export function GuildCarousel({ guilds, basePath = "guilds" }: Props) {
  const plugin = React.useRef(
    Autoplay({ delay: 3500, stopOnInteraction: true })
  )

  return (
    <Carousel
      opts={{ align: "start", loop: true }}
      plugins={[plugin.current]}
      className="w-full"
    >
      <CarouselContent>
        {guilds.map((g) => {
          const shots = g.screenshots ?? []
          const img = g.coverImage ?? shots[0]
          return (
            <CarouselItem key={g.slug} className="basis-full sm:basis-1/2 lg:basis-1/3">
              <a
                href={url(`/${basePath}/${g.slug}`)}
                className="group relative block overflow-hidden rounded-xl ring-1 ring-border aspect-video bg-muted hover:ring-primary transition-all"
                onClick={() => window.umami?.track("guild_click", { name: g.name, rank: g.rank, source: "carousel", type: basePath })}
              >
                {img && (
                  <img
                    src={img}
                    alt={stripGuildShowcase(g.name)}
                    loading="lazy"
                    onError={(e) => ((e.target as HTMLImageElement).style.display = "none")}
                    className="absolute inset-0 h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                  />
                )}
                <div className="absolute inset-0 bg-linear-to-t from-black/70 to-transparent" />
                <div className="absolute bottom-0 left-0 right-0 p-3">
                  <div className="flex items-baseline gap-1.5 mb-1">
                    <p className="text-white font-medium text-sm leading-tight">{stripGuildShowcase(g.name)}</p>
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
      <CarouselPrevious className="-left-4 sm:-left-5" />
      <CarouselNext className="-right-4 sm:-right-5" />
    </Carousel>
  )
}
