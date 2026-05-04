import { useState } from "react"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import TemplateBuilder from "@/components/TemplateBuilder"
import { Terminal, Image } from "lucide-react"

type Mode = "guild" | "solo"

function StepNum({ n, color }: { n: string; color: string }) {
  return (
    <Badge
      variant="secondary"
      className={`shrink-0 h-7 w-7 justify-center rounded-full text-xs font-semibold ${color}`}
    >
      {n}
    </Badge>
  )
}

function GuildPostInstructions() {
  return (
    <div className="space-y-3">
      <div className="rounded-xl border border-blue-500/30 bg-blue-500/5 px-4 py-3 space-y-1.5">
        <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
          <Terminal className="size-3.5 text-blue-500 shrink-0" />
          Use <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">/submit-guild</code>
          <span className="text-xs font-normal text-blue-500 ml-1">easiest</span>
        </p>
        <p className="text-sm text-muted-foreground">
          Type <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">/submit-guild</code> anywhere on the server. Fill in the form and the bot sends you a ready-to-paste post via DM. Go to <span className="font-medium text-foreground">#guild-base-showcase</span>, create a new post, paste it, and attach your screenshots.
        </p>
      </div>
      <div className="flex items-center gap-3">
        <div className="h-px flex-1 bg-border" />
        <span className="text-xs font-semibold text-foreground uppercase tracking-widest">or</span>
        <div className="h-px flex-1 bg-border" />
      </div>
      <p className="text-sm text-muted-foreground">
        Go to <span className="font-medium text-foreground">#guild-base-showcase</span>, click <span className="font-medium text-foreground">New Post</span>, and fill in your first message manually using the template below.
      </p>
    </div>
  )
}

function SoloPostInstructions() {
  return (
    <div className="space-y-3">
      <div className="rounded-xl border border-blue-500/30 bg-blue-500/5 px-4 py-3 space-y-1.5">
        <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
          <Image className="size-3.5 text-blue-500 shrink-0" />
          Just write a story and post your screenshots
          <span className="text-xs font-normal text-blue-500 ml-1">easiest</span>
        </p>
        <p className="text-sm text-muted-foreground">
          Go to <span className="font-medium text-foreground">#solo-building-showcase</span>, click <span className="font-medium text-foreground">New Post</span>, write a short story about your build, and attach your screenshots.
        </p>
      </div>
      <div className="flex items-center gap-3">
        <div className="h-px flex-1 bg-border" />
        <span className="text-xs font-semibold text-foreground uppercase tracking-widest">or</span>
        <div className="h-px flex-1 bg-border" />
      </div>
      <div className="rounded-xl border px-4 py-3 space-y-1.5">
        <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
          <Terminal className="size-3.5 text-muted-foreground shrink-0" />
          Use <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">/submit-solo</code>
          <span className="text-xs font-normal text-muted-foreground ml-1">more structured</span>
        </p>
        <p className="text-sm text-muted-foreground">
          Type <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">/submit-solo</code> anywhere on the server. The bot sends a formatted post via DM. Paste it in <span className="font-medium text-foreground">#solo-building-showcase</span> and add your screenshots.
        </p>
      </div>
      <div className="flex items-center gap-3">
        <div className="h-px flex-1 bg-border" />
        <span className="text-xs font-semibold text-foreground uppercase tracking-widest">or</span>
        <div className="h-px flex-1 bg-border" />
      </div>
      <p className="text-sm text-muted-foreground">
        Go to <span className="font-medium text-foreground">#solo-building-showcase</span>, click <span className="font-medium text-foreground">New Post</span>, and fill in your first message manually using the template below.
      </p>
    </div>
  )
}

export default function BuilderGuide() {
  const [mode, setMode] = useState<Mode>(() => {
    if (typeof window !== "undefined") {
      return new URLSearchParams(window.location.search).get("mode") === "solo" ? "solo" : "guild"
    }
    return "guild"
  })

  function switchMode(next: Mode) {
    setMode(next)
    const params = new URLSearchParams(window.location.search)
    params.set("mode", next)
    history.replaceState(null, "", `?${params.toString()}`)
    window.umami?.track("contribute_mode_switch", { mode: next })
  }

  return (
    <div className="space-y-10">
      {/* Mode toggle */}
      <Tabs value={mode} onValueChange={(v) => switchMode(v as Mode)}>
        <TabsList>
          <TabsTrigger value="guild">Guild base</TabsTrigger>
          <TabsTrigger value="solo">Solo build</TabsTrigger>
        </TabsList>
      </Tabs>

      {/* Steps */}
      <ol className="space-y-8">
        {/* Step 1 */}
        <li className="flex gap-4">
          <StepNum n="1" color="bg-blue-500 text-white border-0" />
          <div className="space-y-1.5 pt-0.5">
            <p className="font-medium text-sm">Join the Discord</p>
            <p className="text-sm text-muted-foreground">All submissions go through our Discord server.</p>
            <Button variant="link" size="sm" asChild className="h-auto p-0 mt-1">
              <a href="https://discord.gg/Qygt9u26Bn" target="_blank" rel="noopener noreferrer" onClick={() => window.umami?.track("discord_cta_click")}>
                Join Discord ↗
              </a>
            </Button>
            {mode === "guild" && (
              <p className="text-sm text-muted-foreground">Even if your guild is already registered, posting yourself gives you more visibility and engagement from the community.</p>
            )}
          </div>
        </li>

        {/* Step 2 */}
        <li className="flex gap-4">
          <StepNum n="2" color="bg-violet-500 text-white border-0" />
          <div className="space-y-3 flex-1 pt-0.5">
            <p className="font-medium text-sm">Post your base</p>
            {mode === "guild" ? <GuildPostInstructions /> : <SoloPostInstructions />}
          </div>
        </li>

        {/* Step 3 */}
        <li className="flex gap-4">
          <StepNum n="3" color="bg-emerald-500 text-white border-0" />
          <div className="space-y-1.5 pt-0.5">
            <p className="font-medium text-sm">Wait for the next sync</p>
            <p className="text-sm text-muted-foreground">The site syncs several times a day. Your {mode === "guild" ? "guild" : "build"} will appear automatically after the next sync.</p>
          </div>
        </li>
      </ol>

      {/* Template builder */}
      <div className="space-y-3 pt-2 border-t border-border">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">First post template</p>
          <p className="text-sm text-muted-foreground mt-1">Fill in the fields below and copy the result into your Discord post.</p>
        </div>
        <TemplateBuilder mode={mode} onModeChange={switchMode} />
        <p className="text-xs text-muted-foreground">Edit your posts at any time. Changes are picked up on the next sync.</p>
      </div>
    </div>
  )
}
