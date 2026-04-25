export function parseLastModified(s: string | undefined): number {
  if (!s) { return 0 }
  const d = new Date(s.replace(" at ", " "))
  return isNaN(d.getTime()) ? 0 : d.getTime()
}

export function relativeTime(ms: number): string {
  const diff = Date.now() - ms
  const mins = Math.floor(diff / 60000)
  if (mins < 60) { return mins <= 1 ? "just now" : `${mins}m ago` }
  const hours = Math.floor(mins / 60)
  if (hours < 24) { return `${hours}h ago` }
  const days = Math.floor(hours / 24)
  if (days < 30) { return `${days}d ago` }
  const months = Math.floor(days / 30)
  if (months < 12) { return `${months}mo ago` }
  return `${Math.floor(months / 12)}y ago`
}

export function formatLastModified(s: string | undefined): { relative: string; full: string } | null {
  if (!s) { return null }
  const d = new Date(s.replace(" at ", " "))
  if (isNaN(d.getTime())) { return null }
  return {
    relative: relativeTime(d.getTime()),
    full: d.toLocaleString("en-US", { month: "short", day: "numeric", year: "numeric", hour: "2-digit", minute: "2-digit", hour12: false }),
  }
}
