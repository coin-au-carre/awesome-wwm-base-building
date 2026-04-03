import { useState } from "react"
import { Button } from "@/components/ui/button"

const PLACEHOLDERS = {
  guildName: "YourGuildName",
  guildId: "12345678",
  builders: "BuilderOne, BuilderTwo",
  lore: "Write your base's story here.",
  whatToVisit: "- Point of interest 1\n- Point of interest 2",
}

function buildTemplate(f: typeof PLACEHOLDERS) {
  const titleId = f.guildId ? ` [${f.guildId}]` : ""
  return [
    `🏯 ${f.guildName || PLACEHOLDERS.guildName}${titleId}`,
    `👷 Builders: ${f.builders || PLACEHOLDERS.builders}`,
    ``,
    `📝 Lore`,
    f.lore,
    ``,
    `🧙 What to visit`,
    f.whatToVisit,
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

const inputClass =
  "w-full rounded-lg border border-border bg-muted/40 px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-ring/50 focus:border-ring transition-colors"

export default function TemplateBuilder() {
  const [fields, setFields] = useState({
    guildName: "",
    guildId: "",
    builders: "",
    lore: "",
    whatToVisit: "",
  })
  const [copied, setCopied] = useState(false)

  const template = buildTemplate(fields)

  function set(key: keyof typeof fields) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setFields((f) => ({ ...f, [key]: e.target.value }))
  }

  async function copy() {
    await navigator.clipboard.writeText(template)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-6">

      <div className="space-y-3 rounded-xl border border-border px-4 py-4">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Fill in your details</p>
        <div className="grid gap-3">
          <div className="grid grid-cols-[1fr_auto] gap-3 items-end">
            <Field label="Guild name">
              <input
                className={inputClass}
                placeholder={PLACEHOLDERS.guildName}
                value={fields.guildName}
                onChange={set("guildName")}
              />
            </Field>
            <Field label="Guild ID" hint="optional">
              <input
                className={inputClass + " w-32"}
                placeholder={PLACEHOLDERS.guildId}
                value={fields.guildId}
                onChange={set("guildId")}
                inputMode="numeric"
              />
            </Field>
          </div>

          <Field label="Builders" hint="in-game names, comma-separated">
            <input
              className={inputClass}
              placeholder={PLACEHOLDERS.builders}
              value={fields.builders}
              onChange={set("builders")}
            />
          </Field>

          <Field label="Lore" hint="optional">
            <textarea
              className={inputClass + " resize-none"}
              rows={3}
              placeholder={PLACEHOLDERS.lore}
              value={fields.lore}
              onChange={set("lore")}
            />
          </Field>

          <Field label="What to visit" hint="optional">
            <textarea
              className={inputClass + " resize-none"}
              rows={3}
              placeholder={PLACEHOLDERS.whatToVisit}
              value={fields.whatToVisit}
              onChange={set("whatToVisit")}
            />
          </Field>
        </div>
      </div>

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
          Paste this as your first post in the guild base forum thread.
        </p>
      </div>

    </div>
  )
}
