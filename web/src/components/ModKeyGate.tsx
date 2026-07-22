import * as React from "react"
import { useEffect, useState } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { WBM_RELAY_URL, modCheckUrl } from "@/lib/gallery"

// sessionStorage (not localStorage) — the key shouldn't outlive the
// browser tab on a shared/public machine. Cleared automatically on
// close, unlike localStorage which would persist indefinitely.
const STORAGE_KEY = "wbm_mod_key"
export const MOD_KEY_EVENT = "wbm-mod-key-changed"
// Matches the id on copyright-watch.astro's content wrapper — the whole
// page (not just MonitorEntry's mod-only panel) stays hidden until a
// key validates against wbm-relay's MOD_SECRET.
const CONTENT_ID = "watch-content"

// MonitorEntry instances are independent client:visible React roots —
// they don't share context, so a plain sessionStorage + window event is
// the simplest way to notify every mounted card when the key changes,
// without wiring a shared provider across separate hydration roots.
export function readModKey(): string {
  if (typeof window === "undefined") return ""
  return sessionStorage.getItem(STORAGE_KEY) || ""
}

function revealContent() {
  document.getElementById(CONTENT_ID)?.classList.remove("hidden")
}

function hideContent() {
  document.getElementById(CONTENT_ID)?.classList.add("hidden")
}

// Full-page key gate for copyright-watch.astro. Unlike an earlier
// version of this component, entering a key now actually validates it
// against wbm-relay's GET /api/mod/check before revealing anything —
// the whole watchlist (not just PII) stays hidden until that check
// passes, not just an opt-in extra panel.
export function ModKeyGate() {
  const [key, setKey] = useState("")
  const [unlocked, setUnlocked] = useState(false)
  const [checking, setChecking] = useState(true)
  const [error, setError] = useState(false)

  useEffect(() => {
    const stored = readModKey()
    if (!stored || !WBM_RELAY_URL) {
      setChecking(false)
      return
    }
    fetch(modCheckUrl(stored))
      .then((res) => {
        if (res.ok) {
          setUnlocked(true)
          revealContent()
        } else {
          sessionStorage.removeItem(STORAGE_KEY)
        }
      })
      .catch(() => {})
      .finally(() => setChecking(false))
  }, [])

  function unlock() {
    setError(false)
    fetch(modCheckUrl(key))
      .then((res) => {
        if (!res.ok) throw new Error("invalid key")
        sessionStorage.setItem(STORAGE_KEY, key)
        window.dispatchEvent(new CustomEvent(MOD_KEY_EVENT))
        setUnlocked(true)
        revealContent()
      })
      .catch(() => setError(true))
  }

  function lock() {
    sessionStorage.removeItem(STORAGE_KEY)
    window.dispatchEvent(new CustomEvent(MOD_KEY_EVENT))
    setKey("")
    setUnlocked(false)
    hideContent()
  }

  if (checking) return null

  if (unlocked) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <span>Unlocked for this tab.</span>
        <Button variant="outline" size="sm" onClick={lock}>
          Lock
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Input
          type="password"
          placeholder="Mod key"
          value={key}
          onChange={(e) => setKey(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && unlock()}
          className="max-w-xs"
        />
        <Button variant="outline" size="sm" onClick={unlock} disabled={!key}>
          Unlock
        </Button>
      </div>
      {error && <p className="text-sm text-rose-500">Wrong key.</p>}
    </div>
  )
}
