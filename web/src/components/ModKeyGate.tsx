import * as React from "react"
import { useState } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"

// sessionStorage (not localStorage) — the key shouldn't outlive the
// browser tab on a shared/public machine. Cleared automatically on
// close, unlike localStorage which would persist indefinitely.
const STORAGE_KEY = "wbm_mod_key"
export const MOD_KEY_EVENT = "wbm-mod-key-changed"

// MonitorEntry instances are independent client:visible React roots —
// they don't share context, so a plain sessionStorage + window event is
// the simplest way to notify every mounted card when the key changes,
// without wiring a shared provider across separate hydration roots.
export function readModKey(): string {
  if (typeof window === "undefined") return ""
  return sessionStorage.getItem(STORAGE_KEY) || ""
}

// One-line unlock control for copyright-watch.astro. Entering a key
// doesn't validate anything client-side — it's just stored and broadcast;
// each MonitorEntry finds out whether it's actually correct from its own
// GET /api/mod/player call (401 vs 200).
export function ModKeyGate() {
  const [key, setKey] = useState("")
  const [unlocked, setUnlocked] = useState(() => readModKey() !== "")

  function unlock() {
    sessionStorage.setItem(STORAGE_KEY, key)
    window.dispatchEvent(new CustomEvent(MOD_KEY_EVENT))
    setUnlocked(true)
  }

  function lock() {
    sessionStorage.removeItem(STORAGE_KEY)
    window.dispatchEvent(new CustomEvent(MOD_KEY_EVENT))
    setKey("")
    setUnlocked(false)
  }

  if (unlocked) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <span>Mod detail unlocked for this tab.</span>
        <Button variant="outline" size="sm" onClick={lock}>
          Lock
        </Button>
      </div>
    )
  }

  return (
    <div className="flex items-center gap-2">
      <Input
        type="password"
        placeholder="Mod key (optional)"
        value={key}
        onChange={(e) => setKey(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && unlock()}
        className="max-w-xs"
      />
      <Button variant="outline" size="sm" onClick={unlock} disabled={!key}>
        Unlock
      </Button>
    </div>
  )
}
