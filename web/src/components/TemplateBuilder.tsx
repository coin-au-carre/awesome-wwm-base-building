import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"

type Mode = "guild" | "solo"

const CONFIG = {
  guild: {
    primaryLabel: "Guild name",
    primaryPlaceholder: "YourGuildName",
    idLabel: "Guild ID",
    idPlaceholder: "12345678",
    buildersPlaceholder: "BuilderOne, BuilderTwo",
    lorePlaceholder: "Write your base's story here.",
    whatToVisitPlaceholder: "- Point of interest 1\n- Point of interest 2",
    emoji: "🏯",
  },
  solo: {
    primaryLabel: "Work label",
    primaryPlaceholder: "My Build Name",
    idLabel: "Builder ID",
    idPlaceholder: "12345678",
    buildersPlaceholder: "YourName",
    lorePlaceholder: "Write your build's story here.",
    whatToVisitPlaceholder: "- Point of interest 1\n- Point of interest 2",
    emoji: "🏠",
  },
}

function buildTemplate(
  fields: { primaryName: string; primaryId: string; builders: string; lore: string; whatToVisit: string },
  mode: Mode,
) {
  const c = CONFIG[mode]
  const idPart = fields.primaryId ? ` [${fields.primaryId}]` : ""
  return [
    `${c.emoji} ${fields.primaryName || c.primaryPlaceholder}${idPart}`,
    ``,
    `👷 Builders: ${fields.builders}`,
    ``,
    `📝 Lore`,
    fields.lore,
    ``,
    `🧙 What to visit`,
    fields.whatToVisit,
    ``,
    `🗳️ Vote with reactions:`,
    `⭐ Best overall | 👍 Good base | 🔥 Amazing creativity`,
  ].join("\n")
}

function Field({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1">
      <div className="flex items-baseline gap-2">
        <label className="text-xs font-medium text-foreground">{label}</label>
        {hint && <span className="text-xs text-muted-foreground">{hint}</span>}
      </div>
      {children}
    </div>
  )
}

const textareaClass =
  "w-full rounded-lg border border-border bg-muted/40 px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-ring/50 focus:border-ring transition-colors resize-none"

export default function TemplateBuilder({ initialMode = "guild" }: { initialMode?: Mode }) {
  const [mode, setMode] = useState<Mode>(initialMode)
  const [fields, setFields] = useState({
    primaryName: "",
    primaryId: "",
    builders: "",
    lore: "",
    whatToVisit: "",
  })
  const [copied, setCopied] = useState(false)

  const c = CONFIG[mode]
  const template = buildTemplate(fields, mode)

  function set(key: keyof typeof fields) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setFields((f) => ({ ...f, [key]: e.target.value }))
  }

  function switchMode(next: Mode) {
    setMode(next)
    setFields({ primaryName: "", primaryId: "", builders: "", lore: "", whatToVisit: "" })
  }

  async function copy() {
    await navigator.clipboard.writeText(template)
    setCopied(true)
    ;(window as any).umami?.track("template_copied", { mode })
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Tabs value={mode} onValueChange={(v) => switchMode(v as Mode)} className="space-y-6">
      <TabsList>
        <TabsTrigger value="guild">Guild builder</TabsTrigger>
        <TabsTrigger value="solo">Solo builder</TabsTrigger>
      </TabsList>

      <Card>
        <CardContent className="space-y-3">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Fill in your details</p>
          <div className="grid gap-3">
            <div className="grid grid-cols-[1fr_auto] gap-3 items-end">
              <Field label={c.primaryLabel}>
                <Input placeholder={c.primaryPlaceholder} value={fields.primaryName} onChange={set("primaryName")} />
              </Field>
              <Field label={c.idLabel} hint="optional">
                <Input className="w-32" placeholder={c.idPlaceholder} value={fields.primaryId} onChange={set("primaryId")} inputMode="numeric" />
              </Field>
            </div>

            <Field label="Builders" hint="in-game names, comma-separated">
              <Input placeholder={c.buildersPlaceholder} value={fields.builders} onChange={set("builders")} />
            </Field>

            <Field label="Lore" hint="optional">
              <textarea
                className={textareaClass}
                rows={3}
                placeholder={c.lorePlaceholder}
                value={fields.lore}
                onChange={set("lore")}
              />
            </Field>

            <Field label="What to visit" hint="optional">
              <textarea
                className={textareaClass}
                rows={3}
                placeholder={c.whatToVisitPlaceholder}
                value={fields.whatToVisit}
                onChange={set("whatToVisit")}
              />
            </Field>
          </div>
        </CardContent>
      </Card>

      <div className="space-y-2">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Your template</p>
        <div className="relative">
          <pre className="rounded-xl bg-muted/60 ring-1 ring-border px-5 py-4 text-xs leading-relaxed overflow-x-auto pr-24">
            <code>{template}</code>
          </pre>
          <Button
            size="sm"
            variant="outline"
            onClick={copy}
            className="absolute top-3 right-3"
          >
            {copied ? "Copied!" : "Copy"}
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">
          Paste this as your first post in the Discord forum thread.
        </p>
      </div>

    </Tabs>
  )
}
