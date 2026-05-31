import { useState, useMemo } from "react"
import * as React from "react"
import type { Bug } from "@/components/BugsTable"
import { cn } from "@/lib/utils"

const EMAIL = "wherewindsmeet@global.netease.com"
const SUBJECT = "Bug Report - Construction/housing/building issues"

const SEVERITY_STYLES: Record<string, string> = {
  high:   "bg-red-500/10 text-red-600 dark:text-red-400",
  normal: "bg-yellow-500/10 text-yellow-600 dark:text-yellow-400",
  low:    "bg-muted text-muted-foreground",
}

function isMobileOnly(bug: Bug) {
  return bug.mobile && !bug.pc && !bug.ps5
}

function buildTemplate(bugs: Bug[], selected: Set<number>, uid: string, platform: string): string {
  const chosen = bugs.filter((_, i) => selected.has(i))
  if (chosen.length === 0) return ""

  const high = chosen.filter((b) => b.severity === "high")
  const others = chosen.filter((b) => b.severity !== "high")

  const lines: string[] = [
    `Issue type: Bug`,
    `Issue: Construction/housing/building issues`,
    `UID: ${uid.trim() || "<YourUID>"}`,
    `Platform: ${platform || "<PC / Mobile / PS5>"}`,
    "",
  ]

  const addSection = (title: string, bugs: Bug[]) => {
    if (bugs.length === 0) return
    lines.push(`## ${title}`, "")
    for (const b of bugs) {
      const detail = b.details ? ` ${b.details}` : ""
      lines.push(`- ${b.title}.${detail}`)
    }
    lines.push("")
  }

  addSection("Critical bugs (all platforms and devices)", high.filter((b) => !isMobileOnly(b)))
  addSection("Critical bugs (mobile only)", high.filter(isMobileOnly))
  addSection("Other bugs (all platforms and devices)", others.filter((b) => !isMobileOnly(b)))
  addSection("Other bugs (mobile only)", others.filter(isMobileOnly))

  return lines.join("\n").trimEnd()
}

export function BugReporter({ bugs }: { bugs: Bug[] }) {
  const reportable = useMemo(() => bugs.filter((b) => b.severity !== "fixed" && b.title), [bugs])

  const [selected, setSelected] = useState<Set<number>>(() => new Set(reportable.map((_, i) => i)))
  const [uid, setUid] = useState("")
  const [platform, setPlatform] = useState("")
  const [copied, setCopied] = useState(false)
  const [open, setOpen] = useState(true)

  function toggleBug(i: number) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(i)) next.delete(i)
      else next.add(i)
      return next
    })
  }

  function toggleAll() {
    setSelected((prev) =>
      prev.size === reportable.length ? new Set() : new Set(reportable.map((_, i) => i))
    )
  }

  const template = useMemo(
    () => buildTemplate(reportable, selected, uid, platform),
    [reportable, selected, uid, platform]
  )

  const fullTemplate = `To: ${EMAIL}\nSubject: ${SUBJECT}\n\n${template}`

  async function copy() {
    await navigator.clipboard.writeText(fullTemplate)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
    window.umami?.track("bugs_report_copy")
  }

  return (
    <div className="rounded-xl border border-primary/20 bg-primary/5 overflow-hidden">
      <button
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center justify-between px-5 py-4 text-left hover:bg-primary/10 transition-colors"
        data-umami-event="bugs_report_open"
      >
        <div>
          <p className="font-semibold text-sm flex items-center gap-2">
            <svg className="size-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <rect width="20" height="16" x="2" y="4" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/>
            </svg>
            Build a report email
          </p>
          <p className="text-xs text-muted-foreground mt-0.5 ml-6">Select bugs, add your UID, copy a ready-to-send template</p>
        </div>
        <svg
          className={cn("size-4 text-muted-foreground transition-transform shrink-0", open && "rotate-180")}
          fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}
        >
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </button>

      {open && (
        <div className="border-t border-border px-5 py-5 space-y-5">
          {/* Bug selection */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Select bugs to include</p>
              <button onClick={toggleAll} className="text-xs text-muted-foreground hover:text-foreground transition-colors">
                {selected.size === reportable.length ? "Deselect all" : "Select all"}
              </button>
            </div>
            <div className="space-y-1.5">
              {reportable.map((bug, i) => (
                <label
                  key={i}
                  className="flex items-start gap-2.5 rounded-lg border border-border px-3 py-2 cursor-pointer hover:bg-muted/40 transition-colors"
                >
                  <input
                    type="checkbox"
                    checked={selected.has(i)}
                    onChange={() => toggleBug(i)}
                    className="mt-0.5 shrink-0 accent-primary"
                  />
                  <div className="min-w-0">
                    <span className="text-sm text-foreground leading-snug">{bug.title}</span>
                    {bug.severity && (
                      <span className={cn("ml-2 inline-block rounded px-1.5 py-0.5 text-[10px] font-medium capitalize", SEVERITY_STYLES[bug.severity])}>
                        {bug.severity}
                      </span>
                    )}
                    {bug.details && <p className="text-xs text-muted-foreground mt-0.5 leading-relaxed">{bug.details}</p>}
                  </div>
                </label>
              ))}
            </div>
          </div>

          {/* UID + Platform */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Your UID</label>
              <input
                type="text"
                placeholder="e.g. 123456789"
                value={uid}
                onChange={(e) => setUid(e.target.value)}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Your Platform</label>
              <select
                value={platform}
                onChange={(e) => setPlatform(e.target.value)}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
              >
                <option value="">Select platform…</option>
                <option value="PC">PC</option>
                <option value="Mobile">Mobile</option>
                <option value="PS5">PS5</option>
                <option value="PC + Mobile">PC + Mobile</option>
                <option value="All platforms">All platforms</option>
              </select>
            </div>
          </div>

          {/* Template preview */}
          {template ? (
            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Template preview</p>
              <div className="relative rounded-lg border border-border bg-muted/30 p-4">
                <pre className="text-xs text-muted-foreground whitespace-pre-wrap leading-relaxed font-mono">
                  <span className="text-foreground/50">To: {EMAIL}</span>{"\n"}
                  <span className="text-foreground/50">Subject: {SUBJECT}</span>{"\n\n"}
                  {template}
                </pre>
                <button
                  onClick={copy}
                  className={cn(
                    "absolute top-3 right-3 flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-semibold transition-colors",
                    copied
                      ? "border-green-500/40 bg-green-500/15 text-green-600 dark:text-green-400"
                      : "border-primary/30 bg-primary/10 text-primary hover:bg-primary/20"
                  )}
                >
                  {copied ? (
                    <><svg className="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5}><polyline points="20 6 9 17 4 12"/></svg> Copied!</>
                  ) : (
                    <><svg className="size-3" viewBox="0 0 256 256" fill="currentColor"><path d="M216,32H88a8,8,0,0,0-8,8V80H40a8,8,0,0,0-8,8V216a8,8,0,0,0,8,8H168a8,8,0,0,0,8-8V176h40a8,8,0,0,0,8-8V40A8,8,0,0,0,216,32ZM160,208H48V96H160Zm48-48H176V88a8,8,0,0,0-8-8H96V48H208Z"/></svg> Copy</>
                  )}
                </button>
              </div>
              <p className="text-xs text-muted-foreground">
                Send to <a href={`mailto:${EMAIL}?subject=${encodeURIComponent(SUBJECT)}`} className="text-foreground underline underline-offset-2 hover:text-primary transition-colors">{EMAIL}</a>
              </p>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground text-center py-4">Select at least one bug to generate the template.</p>
          )}
        </div>
      )}
    </div>
  )
}
