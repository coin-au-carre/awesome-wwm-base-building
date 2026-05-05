import { useState, useEffect, useCallback } from "react"
import { motion, AnimatePresence } from "motion/react"
import { Button } from "@/components/ui/button"
import { ArrowsClockwiseIcon, DiceFiveIcon } from "@phosphor-icons/react"
import type { RankedGuild } from "@/types/guild"
import { formatBuilderName, stripGuildShowcase } from "@/lib/format"
import { url } from "@/lib/url"

interface Props {
  solos: RankedGuild[]
  count?: number
}

function pickRandom<T>(arr: T[], n: number): T[] {
  const copy = [...arr]
  for (let i = copy.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[copy[i], copy[j]] = [copy[j], copy[i]]
  }
  return copy.slice(0, n)
}

interface Tile {
  solo: RankedGuild
  img: string
  key: string
}

function buildTiles(solos: RankedGuild[], count: number): Tile[] {
  return pickRandom(solos, count).map((solo) => {
    const shots = [
      ...(solo.coverImage ? [solo.coverImage] : []),
      ...(solo.screenshots ?? []),
    ]
    const img = shots[Math.floor(Math.random() * shots.length)]
    return { solo, img, key: `${solo.slug}-${img}-${Math.random()}` }
  })
}

export function SoloWall({ solos, count = 12 }: Props) {
  const [tiles, setTiles] = useState<Tile[]>([])
  const [mounted, setMounted] = useState(false)

  const shuffle = useCallback(() => {
    setTiles(buildTiles(solos, count))
    window.umami?.track("solo_wall_shuffle")
  }, [solos, count])

  useEffect(() => {
    setTiles(buildTiles(solos, count))
    setMounted(true)
  }, [])

  if (!mounted) {
    return (
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2">
        {Array.from({ length: count }).map((_, i) => (
          <div key={i} className="aspect-video rounded-xl bg-muted animate-pulse" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="font-heading text-base font-medium flex items-center gap-2">
          <DiceFiveIcon weight="duotone" className="size-4 text-violet-400" />
          Discover Solo Builds
        </h2>
        <Button variant="outline" size="sm" onClick={shuffle} className="gap-2">
          <ArrowsClockwiseIcon className="size-3.5" />
          Shuffle
        </Button>
      </div>
      <AnimatePresence mode="wait">
        <motion.div
          key={tiles.map((t) => t.key).join(",")}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.25 }}
          className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2"
        >
          {tiles.map((tile, idx) => (
            <motion.a
              key={tile.key}
              href={url(`/solos/${tile.solo.slug}`)}
              initial={{ opacity: 0, scale: 0.97 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ delay: idx * 0.025, duration: 0.3, ease: "easeOut" }}
              className="group relative overflow-hidden rounded-xl ring-1 ring-border aspect-video bg-muted hover:ring-primary transition-all"
              onClick={() =>
                window.umami?.track("solo_click", {
                  name: tile.solo.name,
                  source: "wall",
                })
              }
            >
              {tile.img && (
                <img
                  src={tile.img}
                  alt={stripGuildShowcase(tile.solo.name)}
                  loading="lazy"
                  onError={(e) => ((e.target as HTMLImageElement).style.display = "none")}
                  className="absolute inset-0 h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                />
              )}
              <div className="absolute inset-0 bg-linear-to-t from-black/70 via-transparent to-transparent" />
              <div className="absolute bottom-0 left-0 right-0 p-2">
                <p className="text-white font-medium text-xs leading-tight truncate">
                  {stripGuildShowcase(tile.solo.name)}
                </p>
                {tile.solo.builders && tile.solo.builders.length > 0 && (
                  <p className="text-white/60 text-[10px] leading-tight truncate">
                    {tile.solo.builders.map(formatBuilderName).filter(Boolean).join(", ")}
                  </p>
                )}
              </div>
            </motion.a>
          ))}
        </motion.div>
      </AnimatePresence>
    </div>
  )
}
