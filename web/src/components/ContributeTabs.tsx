import { Tabs } from "radix-ui"
import TemplateBuilder from "@/components/TemplateBuilder"

function Step({
  n,
  title,
  body,
  cta,
}: {
  n: string
  title: string
  body: React.ReactNode
  cta?: { label: string; href: string }
}) {
  return (
    <li className="flex gap-4">
      <span className="shrink-0 flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-primary text-xs font-semibold">
        {n}
      </span>
      <div className="space-y-1">
        <p className="font-medium text-sm">{title}</p>
        <p className="text-sm text-muted-foreground">{body}</p>
        {cta && (
          <a
            href={cta.href}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex mt-1 items-center gap-1 text-sm text-primary underline hover:no-underline transition-colors"
          >
            {cta.label}
          </a>
        )}
      </div>
    </li>
  )
}

export default function ContributeTabs() {
  const p = new URLSearchParams(window.location.search).get("mode")
  const initialMode: "guild" | "solo" = p === "solo" ? "solo" : "guild"
  return (
    <Tabs.Root defaultValue="submit" className="space-y-6">
      <Tabs.List className="flex gap-1 rounded-xl bg-muted/50 p-1 w-fit">
        {[
          { value: "submit", label: "Builders" },
          { value: "vote", label: "Construction lovers" },
          { value: "oss", label: "Developers" },
        ].map((tab) => (
          <Tabs.Trigger
            key={tab.value}
            value={tab.value}
            className="rounded-lg px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm cursor-pointer"
          >
            {tab.label}
          </Tabs.Trigger>
        ))}
      </Tabs.List>

      <Tabs.Content value="submit" className="space-y-6">
        <ol className="space-y-5">
          <Step
            n="1"
            title="Join the Discord"
            body={<>All submissions go through our Discord server. <a href="https://discord.gg/Qygt9u26Bn" target="_blank" rel="noopener noreferrer" className="text-primary underline hover:no-underline transition-colors">Join Discord ↗</a></>}
          />
          <Step
            n="2"
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
            title="Wait for the next sync"
            body="The site syncs ~4 times a day. Your guild will appear automatically after the next sync."
          />
        </ol>

        <div className="rounded-lg bg-muted/40 ring-1 ring-border px-4 py-3 space-y-2 text-sm text-muted-foreground">
          <p><span className="font-medium text-foreground">Multiple guilds?</span> Submit each one in a separate thread, all your work is welcome.</p>
          <p><span className="font-medium text-foreground">Image limit:</span> Up to 40 images total, spread across posts. 10 per post is recommended to avoid performance issues.</p>
          <p><span className="font-medium text-foreground">Builder role:</span> Granted automatically when you post a thread.</p>
        </div>
      </Tabs.Content>

      <Tabs.Content value="vote" className="space-y-4">
        <p className="text-sm text-muted-foreground leading-relaxed">
          React to threads in the Discord forum to vote for the builds you love. Votes shape the rankings on the showcase and help the best bases rise to the top.
        </p>
        <p className="text-sm text-muted-foreground leading-relaxed">
          Use ⭐ for 2 points, or 👍 🔥 for 1 point each. The more threads you vote on, the more weight your votes carry.
        </p>
        <p className="text-sm text-muted-foreground leading-relaxed">
          Vote on enough threads and you'll earn the <span className="font-medium text-foreground">Critic</span> role on Discord.
        </p>
        <a
          href="https://discord.gg/Qygt9u26Bn"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-sm text-primary underline hover:no-underline transition-colors"
        >
          Join Discord ↗
        </a>
      </Tabs.Content>

      <Tabs.Content value="oss" className="space-y-6">
        <p className="text-sm text-muted-foreground leading-relaxed">
          The project is open source. Contributions are welcome, open an issue or a pull request.
        </p>
        <a
          href="https://github.com/coin-au-carre/awesome-wwm-base-building"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-sm text-primary underline hover:no-underline transition-colors"
        >
          View on GitHub ↗
        </a>
      </Tabs.Content>
    </Tabs.Root>
  )
}
