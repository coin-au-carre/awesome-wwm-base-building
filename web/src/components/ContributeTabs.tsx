import { useState } from "react"
import TemplateBuilder from "@/components/TemplateBuilder"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Users, Layers, Image, LayoutList, ShieldCheck, Code2, Map, Star, Hammer, Eye, Sparkles, Terminal } from "lucide-react"
import { url } from "@/lib/url"
import { cn } from "@/lib/utils"

function ApiPreview({ src }: { src: string }) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="group relative w-64 overflow-hidden rounded-xl ring-1 ring-border focus:outline-none focus:ring-2 focus:ring-primary"
      >
        <img
          src={src}
          alt="wwmchill — example integration of WBM guild data"
          className="w-full object-cover aspect-video transition-transform duration-300 group-hover:scale-105"
        />
        <div className="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors flex items-center justify-center">
          <svg className="opacity-0 group-hover:opacity-100 transition-opacity w-8 h-8 text-white drop-shadow" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" />
          </svg>
        </div>
      </button>
      <p className="text-xs text-muted-foreground">
        a guild website using our WBM guild data — by <span className="text-foreground font-medium">hugnw</span>
      </p>
      {open && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/90 backdrop-blur-sm"
          onClick={() => setOpen(false)}
        >
          <button
            className="absolute top-4 right-4 text-white/70 hover:text-white transition-colors p-2 rounded-full hover:bg-white/10"
            onClick={() => setOpen(false)}
            aria-label="Close"
          >
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
          <div className="flex items-center justify-center w-full h-full px-8 py-12">
            <img
              src={src}
              alt="wwmchill — example integration of WBM guild data"
              className="max-h-full max-w-full rounded-xl object-contain shadow-2xl select-none"
              onClick={() => setOpen(false)}
            />
          </div>
        </div>
      )}
    </>
  )
}

function Step({
  n,
  title,
  body,
  cta,
  badgeClass,
}: {
  n: string
  title: string
  body: React.ReactNode
  cta?: { label: string; href: string }
  badgeClass?: string
}) {
  return (
    <li className="flex gap-4">
      <Badge
        variant="secondary"
        className={`shrink-0 h-7 w-7 justify-center rounded-full text-xs font-semibold ${badgeClass ?? ""}`}
      >
        {n}
      </Badge>
      <div className="space-y-1">
        <p className="font-medium text-sm">{title}</p>
        <p className="text-sm text-muted-foreground">{body}</p>
        {cta && (
          <Button variant="link" size="sm" asChild className="mt-1 h-auto p-0">
            <a href={cta.href} target="_blank" rel="noopener noreferrer">
              {cta.label}
            </a>
          </Button>
        )}
      </div>
    </li>
  )
}

type Role = "builder" | "scout" | "voter" | "dev"

const roles: {
  id: Role
  icon: React.ReactNode
  label: string
  sublabel: string
  discordRole?: string
  roleColor: string
  cardAccent: string
}[] = [
  {
    id: "builder",
    icon: <Hammer className="size-5" />,
    label: "I am a builder",
    sublabel: "Submit your guild or solo base. Write tutorials.",
    discordRole: "Builder",
    roleColor: "text-blue-500",
    cardAccent: "border-blue-500/40 bg-blue-500/5 hover:bg-blue-500/10",
  },
  {
    id: "scout",
    icon: <Map className="size-5" />,
    label: "I love scouting guild bases",
    sublabel: "Discover and share impressive bases",
    discordRole: "Guild Cartographer",
    roleColor: "text-amber-500",
    cardAccent: "border-amber-500/40 bg-amber-500/5 hover:bg-amber-500/10",
  },
  {
    id: "voter",
    icon: <Star className="size-5" />,
    label: "I rate and review builds",
    sublabel: "Vote, leave feedback, and shape the rankings",
    discordRole: "Trusted Eye",
    roleColor: "text-violet-500",
    cardAccent: "border-violet-500/40 bg-violet-500/5 hover:bg-violet-500/10",
  },
  {
    id: "dev",
    icon: <Code2 className="size-5" />,
    label: "I am a developer",
    sublabel: "Contribute to the open source project",
    discordRole: "Developer",
    roleColor: "text-slate-500",
    cardAccent: "border-slate-500/40 bg-slate-500/5 hover:bg-slate-500/10",
  },
]

const VALID_ROLES: Role[] = ["builder", "scout", "voter", "dev"]

export default function ContributeTabs() {
  const params = new URLSearchParams(window.location.search)
  const initialMode: "guild" | "solo" = params.get("mode") === "solo" ? "solo" : "guild"
  const initialRole: Role = VALID_ROLES.includes(params.get("role") as Role)
    ? (params.get("role") as Role)
    : "builder"
  const [selected, setSelected] = useState<Role>(initialRole)

  function select(id: Role) {
    setSelected(id)
    const next = new URLSearchParams(window.location.search)
    next.set("role", id)
    history.replaceState(null, "", `?${next.toString()}`)
    window.umami?.track("contribute_role_select", { role: id })
  }

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 gap-3">
        {roles.map((role) => {
          const isSelected = selected === role.id
          return (
            <button
              key={role.id}
              type="button"
              onClick={() => select(role.id)}
              className={cn(
                "group flex flex-col gap-2 rounded-xl border p-4 text-left transition-all focus:outline-none focus:ring-2 focus:ring-primary",
                isSelected
                  ? role.cardAccent + " ring-1"
                  : "border-border bg-card hover:bg-muted/50",
              )}
            >
              <div className={cn("transition-colors", isSelected ? role.roleColor : "text-muted-foreground group-hover:text-foreground")}>
                {role.icon}
              </div>
              <div className="space-y-0.5">
                <p className={cn("text-sm font-semibold leading-tight", isSelected ? "text-foreground" : "text-foreground/80")}>
                  {role.label}
                </p>
                <p className="text-xs text-muted-foreground leading-snug">{role.sublabel}</p>
              </div>
              {role.discordRole && (
                <Badge
                  variant="secondary"
                  className={cn(
                    "w-fit text-xs transition-colors",
                    isSelected ? role.roleColor : "text-muted-foreground",
                  )}
                >
                  {role.discordRole}
                </Badge>
              )}
            </button>
          )
        })}
      </div>

      <div className="space-y-6">
        {selected === "builder" && (
          <>
            <ol className="space-y-5">
              <Step
                n="1"
                badgeClass="bg-blue-500 text-white border-0"
                title="Join the Discord"
                body={<>All submissions go through our Discord server. <Button variant="link" size="sm" asChild className="h-auto p-0"><a href="https://discord.gg/Qygt9u26Bn" target="_blank" rel="noopener noreferrer" onClick={() => window.umami?.track("discord_cta_click")}>Join Discord ↗</a></Button></>}
              />
              <Step
                n="2"
                badgeClass="bg-violet-500 text-white border-0"
                title="Post your base"
                body={
                  <div className="space-y-3 mt-1">
                    <div className="rounded-xl border border-blue-500/30 bg-blue-500/5 px-4 py-3 space-y-1.5">
                      <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                        <Terminal className="size-3.5 text-blue-500 shrink-0" />
                        Use the <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">/submit-guild</code> command <span className="text-xs font-normal text-blue-500 ml-1">easiest</span>
                      </p>
                      <p className="text-sm text-muted-foreground">Type <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">/submit-guild</code> anywhere on the server. Fill in the form and the bot will send you a ready-to-paste formatted post via DM, with the thread title and all your content included. Just go to <span className="font-medium text-foreground">#guild-base-showcase</span>, create a new post, paste it, and add your screenshots.</p>
                    </div>
                    <div className="flex items-center gap-3">
                      <div className="h-px flex-1 bg-border" />
                      <span className="text-xs font-semibold text-foreground uppercase tracking-widest">or</span>
                      <div className="h-px flex-1 bg-border" />
                    </div>
                    <p className="text-sm text-muted-foreground">Go to <span className="font-medium text-foreground">#guild-base-showcase</span> (guild bases) or <span className="font-medium text-foreground">#solo-building-showcase</span> (solo builds), click <span className="font-medium text-foreground">New Post</span>, and fill in your first message manually using the template below.</p>
                  </div>
                }
              />
            </ol>

            <div className="space-y-2">
              <p className="text-sm text-muted-foreground"><strong>First post</strong> template:</p>
              <TemplateBuilder initialMode={initialMode} />
              <p className="text-xs text-muted-foreground">
                Edit your posts at any time. Changes are picked up on the next sync.
              </p>
            </div>

            <ol className="space-y-5">
              <Step
                n="3"
                badgeClass="bg-emerald-500 text-white border-0"
                title="Wait for the next sync"
                body="The site syncs several times a day. Your guild will appear automatically after the next sync."
              />
            </ol>

            <hr className="border-border" />

            <Card>
              <CardContent className="divide-y divide-border p-0">
                {[
                  {
                    icon: <Users className="size-4 text-slate-500 shrink-0 mt-0.5" />,
                    title: "Thread already exists for your guild?",
                    body: "Create a new one and ask a moderator to close the old one.",
                  },
                  {
                    icon: <ShieldCheck className="size-4 text-slate-500 shrink-0 mt-0.5" />,
                    title: "Not comfortable posting?",
                    body: "A moderator can post it for you.",
                  },
                  {
                    icon: <Users className="size-4 text-blue-500 shrink-0 mt-0.5" />,
                    title: "Multiple guilds, one builder?",
                    body: "You can submit in different threads. One guild = one thread.",
                  },
                  {
                    icon: <Layers className="size-4 text-violet-500 shrink-0 mt-0.5" />,
                    title: "One guild, multiple builds?",
                    body: "You can post different styles even if it's the same guild. Same guild = one thread.",
                  },
                  {
                    icon: <Image className="size-4 text-emerald-500 shrink-0 mt-0.5" />,
                    title: "Cover image",
                    body: <>Add <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">Cover: N</code> to your first post to pin screenshot #N as the cover. Example: <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">Cover: 3</code> picks your third image.</>,
                  },
                  {
                    icon: <LayoutList className="size-4 text-amber-500 shrink-0 mt-0.5" />,
                    title: "Screenshot sections",
                    body: <>Organize screenshots into labeled groups by posting a message starting with <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs"># Section name</code> (using <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">#</code>, <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">##</code>, or <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">###</code>). Post the label before or after the images it describes.</>,
                  }
                ].map(({ icon, title, body }) => (
                  <div key={title} className="flex gap-3 px-4 py-3">
                    {icon}
                    <div className="space-y-0.5">
                      <p className="text-sm font-medium text-foreground">{title}</p>
                      <p className="text-sm text-muted-foreground">{body}</p>
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>

            <div className="space-y-1.5">
              <p className="text-sm font-medium text-foreground">Write a tutorial</p>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Know a building trick worth sharing? Write a guide and we'll publish it on the{" "}
                <a href={url("/tutorials")} className="text-foreground underline underline-offset-2 hover:text-primary transition-colors">
                  tutorials page
                </a>.
              </p>
            </div>
          </>
        )}

        {selected === "scout" && (
          <>
            <div className="space-y-1.5">
              <p className="text-sm font-medium text-foreground">Found a base worth sharing?</p>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Use <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">/scout-guild</code> to report it. The command opens a short form: guild name, what to visit. The bot then logs it to{" "}
                <span className="font-medium text-foreground">#guild-discoveries</span> for the community. You can still share screenshots manually to <span className="font-medium text-foreground">#guild-discoveries</span> for passionate discussions! 
              </p>
            </div>
            <div className="space-y-1.5">
              <p className="text-sm font-medium text-foreground">Know the builder? Or make a new friend.</p>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Reach out and encourage them to submit it themselves with{" "}
                <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">/submit-guild</code>. They'll own their thread and can update it anytime: lore, screenshots, cover image, what to visit.
              </p>
            </div>
            <div className="space-y-1.5">
              <p className="text-sm font-medium text-foreground">Earn the Guild Cartographer role</p>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Moderators grant the <span className="font-medium text-foreground">Guild Cartographer</span> role to members who consistently scout and document guild bases across the community.
              </p>
            </div>
          </>
        )}

        {selected === "voter" && (
          <>
            <div className="space-y-1.5">
              <p className="text-sm font-medium text-foreground">React to builds</p>
              <p className="text-sm text-muted-foreground leading-relaxed">
                React to threads in the forum to vote for the builds you love. Use ⭐ for 2 points, or 👍 🔥 for 1 point each. Votes shape the rankings on the showcase and help the best bases rise to the top.
              </p>
            </div>
            <div className="space-y-1.5">
              <p className="text-sm font-medium text-foreground">Leave feedback</p>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Reply inside a thread with genuine feedback on what works well, what could be improved, or what inspired you. Builders appreciate detailed feedback far more than just a reaction.
              </p>
            </div>

            <hr className="border-border" />

            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Star className="size-4 text-amber-500 shrink-0" />
                <p className="text-sm font-semibold text-foreground">Critic</p>
              </div>
              <p className="text-sm text-muted-foreground leading-relaxed">
                The <span className="font-medium text-foreground">Critic</span> role is awarded automatically by the bot to active voters who engage broadly across many guilds. Keep voting consistently and it will come.
              </p>
            </div>

            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Eye className="size-4 text-violet-500 shrink-0" />
                <p className="text-sm font-semibold text-foreground">Trusted Eye</p>
              </div>
              <p className="text-sm text-muted-foreground leading-relaxed">
                The <span className="font-medium text-foreground">Trusted Eye</span> role is granted by moderators to members recognized for their expertise and quality feedback. Not automatic — it carries more voting weight and comes with a few guidelines to keep rankings fair.
              </p>

              <div className="rounded-xl ring-1 ring-border overflow-hidden divide-y divide-border">
                {[
                  { body: "Voting on your own guild counts as a normal reaction and does not apply your Trusted Eye bonus." },
                  { body: "Only react if you've actually looked at the screenshots and visited the guild base. No support votes." },
                  { body: "Avoid coordinating votes with other Trusted Eyes on specific builds." },
                  { body: "Prioritize guilds with no recognition yet. The goal is to surface hidden gems, not just reward popular builders." },
                ].map(({ body }) => (
                  <div key={body} className="flex gap-3 px-4 py-2">
                    <ShieldCheck className="size-4 text-violet-400 shrink-0 mt-0.5" />
                    <p className="text-sm text-muted-foreground">{body}</p>
                  </div>
                ))}
              </div>

              <div className="space-y-1.5">
                <div className="flex items-center gap-1.5">
                  <Sparkles className="size-3.5 text-violet-500 shrink-0" />
                  <p className="text-xs font-semibold text-foreground uppercase tracking-wide">What to look for</p>
                </div>
                <div className="grid grid-cols-1 gap-1">
                  {[
                    ["First impression", "Does it immediately feel impressive or cohesive?"],
                    ["Creativity", "Unexpected themes, layouts, or storytelling through builds."],
                    ["Technique", "Clever use of layering, furniture tricks, or item combinations."],
                    ["Consistency", "Does the style hold up throughout the base, not just in one spot?"],
                    ["Atmosphere", "Lighting, color palette, and spatial flow."],
                    ["Attention to detail", "Small touches that reward a closer look."],
                    ["Theme execution", "How well does the build commit to and deliver on its concept?"],
                  ].map(([label, desc]) => (
                    <div key={label} className="flex gap-2 py-1">
                      <p className="text-xs font-medium text-foreground w-36 shrink-0">{label}</p>
                      <p className="text-xs text-muted-foreground leading-snug">{desc}</p>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </>
        )}

        {selected === "dev" && (
          <>
            <p className="text-sm text-muted-foreground leading-relaxed">
              The project is open source. Contributions are welcome — chat with us, open an issue, or submit a pull request.
            </p>
            <p className="text-sm text-muted-foreground leading-relaxed">
              Ping us on Discord if you want to integrate any work here, we'll be glad to help. You can use the data freely as long as you credit the website and community.
            </p>
            <ApiPreview src={url("/images/wwmchill_api.png")} />
          </>
        )}
      </div>
    </div>
  )
}
