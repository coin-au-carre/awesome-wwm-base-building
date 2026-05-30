import { useState } from "react"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { BlueprintGrid } from "@/components/BlueprintGrid"
import { BlueprintIcon, UserIcon, CaretDownIcon } from "@phosphor-icons/react"
import type { RankedBlueprint } from "@/types/blueprint"
import { url } from "@/lib/url"
import { cn } from "@/lib/utils"

export interface BuilderBlueprintPreview {
  name: string
  slug: string
  coverImage?: string
  isFree?: boolean
  isPayToBuild?: boolean
  price?: string
}

export interface BuilderCardData {
  name: string
  slug: string
  blueprints: BuilderBlueprintPreview[]
}

interface Props {
  blueprints: RankedBlueprint[]
  allTags: string[]
  builders: BuilderCardData[]
}

function BuilderRow({ builder }: { builder: BuilderCardData }) {
  const [open, setOpen] = useState(false)
  const initial = builder.name[0]?.toUpperCase() ?? "?"

  return (
    <div className="rounded-xl ring-1 ring-border bg-card overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center gap-3 px-4 py-3 hover:bg-muted/50 transition-colors text-left cursor-pointer"
        data-umami-event="blueprint_builder_expand"
        data-umami-event-name={builder.name}
      >
        <div className="size-9 rounded-full bg-primary/10 text-primary flex items-center justify-center text-sm font-bold shrink-0 ring-1 ring-primary/20">
          {initial}
        </div>
        <div className="flex-1 min-w-0">
          <p className="font-medium text-sm leading-tight">{builder.name}</p>
          <p className="text-[11px] text-muted-foreground mt-0.5">
            {builder.blueprints.length} blueprint{builder.blueprints.length !== 1 ? "s" : ""}
          </p>
        </div>
        <CaretDownIcon
          weight="bold"
          className={cn("size-4 text-muted-foreground shrink-0 transition-transform duration-200", open && "rotate-180")}
        />
      </button>

      {open && (
        <div className="border-t border-border px-4 py-4 bg-muted/20">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {builder.blueprints.map((bp) => (
              <a
                key={bp.slug}
                href={url(`/blueprints/${bp.slug}`)}
                className="group flex items-center gap-3 rounded-lg ring-1 ring-border bg-card hover:ring-primary transition-all px-3 py-2"
                data-umami-event="blueprint_builder_blueprint_click"
                data-umami-event-name={bp.name}
              >
                <div className="relative shrink-0 w-16 aspect-video rounded-md overflow-hidden bg-muted">
                  {bp.coverImage ? (
                    <img
                      src={bp.coverImage}
                      alt={bp.name}
                      loading="lazy"
                      onError={(e) => { ;(e.currentTarget as HTMLImageElement).style.display = "none" }}
                      className="absolute inset-0 h-full w-full object-cover"
                    />
                  ) : (
                    <div className="absolute inset-0 flex items-center justify-center text-muted-foreground/30 text-lg">📐</div>
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium leading-tight line-clamp-1">{bp.name}</p>
                  <p className="text-[10px] text-muted-foreground mt-0.5">
                    {bp.isFree && bp.isPayToBuild ? "Free + Paid" : bp.isPayToBuild ? bp.price ?? "Pay-to-build" : "Free"}
                  </p>
                </div>
                <svg
                  className="size-3 shrink-0 text-muted-foreground/40 group-hover:text-primary group-hover:translate-x-0.5 transition-all"
                  fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}
                >
                  <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
                </svg>
              </a>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function resolveInitialTab(): string {
  if (typeof window === "undefined") return "blueprints"
  return new URLSearchParams(window.location.search).get("tab") === "builders" ? "builders" : "blueprints"
}

export function BlueprintPageTabs({ blueprints, allTags, builders }: Props) {
  const [tab, setTab] = useState<string>(() => resolveInitialTab())

  function handleTabChange(value: string) {
    setTab(value)
    const u = new URL(window.location.href)
    if (value === "blueprints") {
      u.searchParams.delete("tab")
    } else {
      u.searchParams.set("tab", value)
    }
    history.replaceState(null, "", u.toString())
  }

  return (
    <Tabs value={tab} onValueChange={handleTabChange}>
      <TabsList className="w-full rounded-none border-b border-border bg-transparent p-0 gap-0 h-auto">
        <TabsTrigger
          value="blueprints"
          className="flex-1 sm:flex-none rounded-none border-b-2 border-transparent px-5 py-2.5 text-sm font-medium data-[state=active]:border-primary data-[state=active]:text-foreground data-[state=active]:shadow-none data-[state=active]:bg-transparent gap-2"
        >
          <BlueprintIcon weight="duotone" className="size-4 shrink-0" />
          Blueprints
          {blueprints.length > 0 && (
            <span className="ml-0.5 text-[11px] text-muted-foreground font-normal">{blueprints.length}</span>
          )}
        </TabsTrigger>
        <TabsTrigger
          value="builders"
          className="flex-1 sm:flex-none rounded-none border-b-2 border-transparent px-5 py-2.5 text-sm font-medium data-[state=active]:border-primary data-[state=active]:text-foreground data-[state=active]:shadow-none data-[state=active]:bg-transparent gap-2"
        >
          <UserIcon weight="duotone" className="size-4 shrink-0" />
          Builders
          {builders.length > 0 && (
            <span className="ml-0.5 text-[11px] text-muted-foreground font-normal">{builders.length}</span>
          )}
        </TabsTrigger>
      </TabsList>

      <TabsContent value="blueprints" className="mt-5">
        {blueprints.length === 0 ? (
          <p className="text-sm text-muted-foreground py-8 text-center">No blueprints yet.</p>
        ) : (
          <BlueprintGrid blueprints={blueprints} allTags={allTags} />
        )}
      </TabsContent>

      <TabsContent value="builders" className="mt-5">
        {builders.length === 0 ? (
          <p className="text-sm text-muted-foreground py-8 text-center">No builders yet.</p>
        ) : (
          <div className="flex flex-col gap-2">
            {builders.map((b) => (
              <BuilderRow key={b.slug} builder={b} />
            ))}
          </div>
        )}
      </TabsContent>
    </Tabs>
  )
}
