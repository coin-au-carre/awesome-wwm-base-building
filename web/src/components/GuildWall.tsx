import { useState, useEffect, useCallback } from "react"
import { motion, AnimatePresence } from "motion/react"
import { Button } from "@/components/ui/button"
import { ArrowsClockwiseIcon, DiceFiveIcon } from "@phosphor-icons/react"
import { formatBuilderName, stripGuildShowcase } from "@/lib/format"
import { formatLastModified } from "@/lib/dates"
import { url } from "@/lib/url"

export interface WallGuild {
  slug: string
  name: string
  guildName?: string
  builders: string[]
  lastModified?: string
  img: string
}

interface Props {
  guilds: WallGuild[]
  count?: number
}

function pickRandom<T>(arr: T[], n: number): T[] {
  const copy = [...arr]
  const end = Math.min(n, copy.length)
  for (let i = 0; i < end; i++) {
    const j = i + Math.floor(Math.random() * (copy.length - i))
    ;[copy[i], copy[j]] = [copy[j], copy[i]]
  }
  return copy.slice(0, end)
}

interface Tile {
  guild: WallGuild
  key: string
}

function buildTiles(guilds: WallGuild[], count: number): Tile[] {
  return pickRandom(guilds, count).map((guild) => ({
    guild,
    key: `${guild.slug}-${Math.random()}`,
  }))
}

export function GuildWall({ guilds, count = 12 }: Props) {
  const [tiles, setTiles] = useState<Tile[]>([])
  const [mounted, setMounted] = useState(false)

  const shuffle = useCallback(() => {
    setTiles(buildTiles(guilds, count))
    window.umami?.track("guild_wall_shuffle")
  }, [guilds, count])

  useEffect(() => {
    setTiles(buildTiles(guilds, count))
    setMounted(true)
  }, [])

  if (!mounted) {
    return (
      <div className="flex overflow-x-auto sm:grid sm:overflow-visible gap-2 pb-2 sm:pb-0 sm:grid-cols-3 [scrollbar-width:none] [-webkit-overflow-scrolling:touch]">
        {Array.from({ length: count }).map((_, i) => (
          <div key={i} className="aspect-video rounded-xl bg-muted animate-pulse shrink-0 w-[calc(50vw-20px)] sm:w-auto" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="font-heading text-base font-medium flex items-center gap-2">
          <DiceFiveIcon weight="duotone" className="size-4 text-violet-400" />
          Discover Guild Bases
        </h2>
        <Button variant="outline" size="sm" onClick={shuffle} className="gap-2">
          <ArrowsClockwiseIcon className="size-3.5" />
          Shuffle
        </Button>
      </div>
      <div className="relative">
      <AnimatePresence mode="sync">
        <motion.div
          key={tiles.map((t) => t.key).join(",")}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.25 }}
          className="flex overflow-x-auto sm:grid sm:overflow-visible snap-x snap-mandatory sm:snap-none gap-2 pb-2 sm:pb-0 sm:grid-cols-3 [scrollbar-width:none] [-webkit-overflow-scrolling:touch]"
        >
          {tiles.map((tile, idx) => (
            <motion.a
              key={tile.key}
              href={url(`/guilds/${tile.guild.slug}`)}
              initial={{ opacity: 0, scale: 0.97 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ delay: idx * 0.025, duration: 0.3, ease: "easeOut" }}
              className="group relative overflow-hidden rounded-xl ring-1 ring-border aspect-video bg-muted hover:ring-primary transition-all shrink-0 w-[calc(50vw-20px)] sm:w-auto snap-start"
              onClick={() =>
                window.umami?.track("guild_click", {
                  name: tile.guild.name,
                  source: "wall",
                })
              }
            >
              <img
                src={tile.guild.img}
                alt={stripGuildShowcase(tile.guild.guildName || tile.guild.name)}
                loading="lazy"
                onError={(e) => ((e.target as HTMLImageElement).style.display = "none")}
                className="absolute inset-0 h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
              />
              <div className="absolute inset-0 bg-linear-to-t from-black/70 via-transparent to-transparent" />
              {tile.guild.lastModified && (() => {
                const lm = formatLastModified(tile.guild.lastModified)
                return lm ? (
                  <div
                    className="absolute top-2 left-2 z-20 text-[10px] font-medium px-1.5 py-0.5 rounded-md bg-black/50 backdrop-blur-sm text-white/60"
                    title={lm.full}
                  >
                    {lm.relative}
                  </div>
                ) : null
              })()}
              <div className="absolute bottom-0 left-0 right-0 p-2">
                <p className="text-white font-medium text-xs leading-tight truncate">
                  {stripGuildShowcase(tile.guild.guildName || tile.guild.name)}
                </p>
                {tile.guild.builders && tile.guild.builders.length > 0 && (
                  <p className="text-white/60 text-[10px] leading-tight truncate">
                    {tile.guild.builders.map(formatBuilderName).filter(Boolean).join(", ")}
                  </p>
                )}
              </div>
            </motion.a>
          ))}
        </motion.div>
      </AnimatePresence>
      {/* right-edge fade — signals more cards to swipe on mobile */}
      <div className="absolute right-0 top-0 bottom-2 w-10 bg-linear-to-l from-background to-transparent pointer-events-none sm:hidden" />
      </div>
      <p className="text-[11px] text-muted-foreground/45 sm:hidden mt-0.5 text-right pr-1">swipe to explore →</p>
    </div>
  )
}
