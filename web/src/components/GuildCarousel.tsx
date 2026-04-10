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
import { url } from "@/lib/url"

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
              >
                {img && (
                  <img
                    src={img}
                    alt={g.name}
                    loading="lazy"
                    onError={(e) => ((e.target as HTMLImageElement).style.display = "none")}
                    className="absolute inset-0 h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                  />
                )}
                <div className="absolute inset-0 bg-linear-to-t from-black/70 to-transparent" />
                <div className="absolute bottom-0 left-0 right-0 p-3">
                  <p className="text-white font-medium text-sm leading-tight mb-1">{g.name}</p>
                  {g.tags && g.tags.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {g.tags.slice(0, 3).map((tag) => (
                        <span
                          key={tag}
                          className="inline-flex items-center rounded-full bg-black/40 backdrop-blur-sm px-2 py-0.5 text-[10px] text-white/90"
                        >
                          {tag}
                        </span>
                      ))}
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
