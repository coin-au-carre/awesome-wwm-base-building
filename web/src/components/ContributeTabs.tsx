import TemplateBuilder from "@/components/TemplateBuilder"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Users, Layers, Image, LayoutList, ShieldCheck } from "lucide-react"

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

export default function ContributeTabs() {
  const p = new URLSearchParams(window.location.search).get("mode")
  const initialMode: "guild" | "solo" = p === "solo" ? "solo" : "guild"
  return (
    <Tabs
      defaultValue="submit"
      className="space-y-6"
      onValueChange={(tab) => (window as any).umami?.track("contribute_tab_switch", { tab })}
    >
      <TabsList>
        {[
          { value: "submit", label: "Builders" },
          { value: "vote", label: "Construction lovers" },
          { value: "oss", label: "Developers" },
        ].map((tab) => (
          <TabsTrigger key={tab.value} value={tab.value}>
            {tab.label}
          </TabsTrigger>
        ))}
      </TabsList>

      <TabsContent value="submit" className="space-y-6">
        <ol className="space-y-5">
          <Step
            n="1"
            badgeClass="bg-blue-500 text-white border-0"
            title="Join the Discord"
            body={<>All submissions go through our Discord server. <Button variant="link" size="sm" asChild className="h-auto p-0"><a href="https://discord.gg/Qygt9u26Bn" target="_blank" rel="noopener noreferrer" onClick={() => (window as any).umami?.track("discord_cta_click", { page: "contribute" })}>Join Discord ↗</a></Button></>}
          />
          <Step
            n="2"
            badgeClass="bg-violet-500 text-white border-0"
            title="Post your base"
            body={<>Create a thread in <span className="font-medium text-foreground">#guild-base-showcase</span> or <span className="font-medium text-foreground">#solo-building-showcase</span>. Title it with your guild or build name, then fill in your first post using the template below and attach your screenshots in the post and follow-up posts.</>}
          />
        </ol>

        <div className="space-y-2">
          <p className="text-sm text-muted-foreground">First post template:</p>
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
              },
              {
                icon: <ShieldCheck className="size-4 text-rose-500 shrink-0 mt-0.5" />,
                title: "Builder role",
                body: "Granted automatically when you post a thread.",
              },
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
      </TabsContent>

      <TabsContent value="vote" className="space-y-4">
        <p className="text-sm text-muted-foreground leading-relaxed">
          React to threads in the Discord forum to vote for the builds you love. Votes shape the rankings on the showcase and help the best bases rise to the top.
        </p>
        <p className="text-sm text-muted-foreground leading-relaxed">
          Use ⭐ for 2 points, or 👍 🔥 for 1 point each. The more threads you vote on, the more weight your votes carry.
        </p>
        <p className="text-sm text-muted-foreground leading-relaxed">
          Vote on enough threads and you'll earn the <span className="font-medium text-foreground">Critic</span> role on Discord.
        </p>
        <Button variant="link" size="sm" asChild className="h-auto p-0">
          <a href="https://discord.gg/Qygt9u26Bn" target="_blank" rel="noopener noreferrer" onClick={() => (window as any).umami?.track("discord_cta_click", { page: "contribute" })}>
            Join Discord ↗
          </a>
        </Button>
      </TabsContent>

      <TabsContent value="oss" className="space-y-6">
        <p className="text-sm text-muted-foreground leading-relaxed">
          The project is open source. Contributions are welcome, chat with us, open an issue or a pull request.
        </p>
        <p className="text-sm text-muted-foreground leading-relaxed">
          Ping us on Discord if you want to integrate any work here, we'll be glad to help. You can use it whenever you want as long as you credit the website/community.
        </p>
      </TabsContent>
    </Tabs>
  )
}
