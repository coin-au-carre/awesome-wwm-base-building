import { useState } from "react"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import TemplateBuilder from "@/components/TemplateBuilder"
import { Terminal, Image, LayoutList, Sparkles, Users, Layers, Bold, FileImage, type LucideIcon } from "lucide-react"
import { url } from "@/lib/url"

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

function OptionLabel({ label }: { label: string }) {
  return <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{label}</p>
}

function GuildPostInstructions() {
  return (
    <div className="space-y-4">
      <div className="space-y-1.5">
        <OptionLabel label="Option A — use the bot" />
        <div className="rounded-xl border border-blue-500/30 bg-blue-500/5 px-4 py-3 space-y-1.5">
          <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
            <Terminal className="size-3.5 text-blue-500 shrink-0" />
            Use <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">/submit-guild</code>
            <span className="text-xs font-normal text-blue-500 ml-1">easiest</span>
          </p>
          <p className="text-sm text-muted-foreground">
            Type <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">/submit-guild</code> anywhere on the server. Fill in the form and the bot sends you a ready-to-paste post via DM. Go to <span className="font-medium text-foreground">#guild-base-showcase</span>, create a new post, paste it, and attach your screenshots (10 max for the first post).
          </p>
        </div>
      </div>
      <div className="space-y-1.5">
        <OptionLabel label="Option B — post manually" />
        <div className="rounded-xl border px-4 py-3 space-y-1.5">
          <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
            <LayoutList className="size-3.5 text-muted-foreground shrink-0" />
            Post directly in the forum
          </p>
          <p className="text-sm text-muted-foreground">
            Go to <span className="font-medium text-foreground">#guild-base-showcase</span>, click <span className="font-medium text-foreground">New Post</span>, and fill in your first message using the <a href="#template" className="underline underline-offset-2 hover:text-foreground transition-colors">template below ↓</a>.
          </p>
        </div>
      </div>
    </div>
  )
}

function SoloPostInstructions() {
  return (
    <div className="space-y-4">
      <div className="space-y-1.5">
        <OptionLabel label="Option A — free form" />
        <div className="rounded-xl border border-blue-500/30 bg-blue-500/5 px-4 py-3 space-y-1.5">
          <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
            <Image className="size-3.5 text-blue-500 shrink-0" />
            Just write a story and post your screenshots
            <span className="text-xs font-normal text-blue-500 ml-1">easiest</span>
          </p>
          <p className="text-sm text-muted-foreground">
            Go to <span className="font-medium text-foreground">#solo-building-showcase</span>, click <span className="font-medium text-foreground">New Post</span>, write a short story about your build, and attach your screenshots (10 max for the first post).
          </p>
        </div>
      </div>
      <div className="space-y-1.5">
        <OptionLabel label="Option B — use the bot" />
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
      </div>
      <div className="space-y-1.5">
        <OptionLabel label="Option C — post manually" />
        <div className="rounded-xl border px-4 py-3 space-y-1.5">
          <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
            <LayoutList className="size-3.5 text-muted-foreground shrink-0" />
            Post directly in the forum
          </p>
          <p className="text-sm text-muted-foreground">
            Go to <span className="font-medium text-foreground">#solo-building-showcase</span>, click <span className="font-medium text-foreground">New Post</span>, and fill in your first message using the <a href="#template" className="underline underline-offset-2 hover:text-foreground transition-colors">template below ↓</a>.
          </p>
        </div>
      </div>
    </div>
  )
}

type Tip = { icon: LucideIcon; color: string; title: string; body: React.ReactNode }

const tips: Tip[] = [
  {
    icon: FileImage,
    color: "text-orange-400",
    title: "Screenshot size",
    body: "Keep screenshots under 8 MB for fast loading. Larger files slow down the page for everyone.",
  },
  {
    icon: Image,
    color: "text-emerald-500",
    title: "Cover image",
    body: <>Add <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">Cover: N</code> to your first post to pin screenshot #N as the cover. Example: <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">Cover: 3</code> picks your third image.</>,
  },
  {
    icon: LayoutList,
    color: "text-amber-500",
    title: "Screenshot sections",
    body: <>Organize screenshots into labeled groups by posting a message starting with <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs"># Section name</code>. Post the label before or after the images it describes.</>,
  },
  {
    icon: Bold,
    color: "text-sky-500",
    title: "Text formatting",
    body: <>Use <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">**bold**</code>, <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">*italic*</code>, or <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">~~strikethrough~~</code> in your lore and what-to-visit text. Wrap text in <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">||spoilers||</code> to hide it until clicked.</>,
  },
  {
    icon: Sparkles,
    color: "text-violet-500",
    title: "Want more votes?",
    body: "Update your screenshots or videos. This triggers an announcement in the general chat that re-exposes your base to the community.",
  },
  {
    icon: Users,
    color: "text-slate-400",
    title: "Thread already exists?",
    body: "Create a new one and ask a moderator to close the old one.",
  },
  {
    icon: Users,
    color: "text-slate-400",
    title: "Not comfortable posting?",
    body: "A moderator can post it for you.",
  },
  {
    icon: Layers,
    color: "text-blue-400",
    title: "Multiple guilds, one builder?",
    body: "You can submit in as many threads as you want. One build = one thread.",
  },
  {
    icon: Layers,
    color: "text-blue-400",
    title: "One guild, multiple builds?",
    body: <>Name your threads <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">GuildName - Build Title</code> and they'll be grouped automatically on the leaderboard. <a href={url("/tutorials/multiple-builds-per-guild")} className="underline underline-offset-2 hover:text-foreground transition-colors">Full guide ↗</a></>,
  },
]

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

      {/* Tips & formatting */}
      <div id="tips" className="space-y-5 border-t border-border pt-2">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Tips & formatting</p>
          <p className="text-sm text-muted-foreground mt-1">Everything you can do to improve your entry.</p>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {tips.map(({ icon: Icon, color, title, body }) => (
            <div key={title} className="rounded-xl border border-border bg-card p-4 space-y-2">
              <div className="flex items-center gap-2">
                <Icon size={15} className={color} />
                <p className="text-sm font-medium text-foreground">{title}</p>
              </div>
              <p className="text-sm text-muted-foreground leading-relaxed">{body}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Template builder */}
      <div id="template" className="space-y-3 pt-2 border-t border-border">
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
