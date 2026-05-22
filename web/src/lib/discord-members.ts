let cached: number | null = null
let fetched = false

export async function getDiscordMemberCount(): Promise<number | null> {
  if (fetched) return cached
  fetched = true
  try {
    const res = await fetch("https://discord.com/api/invites/Qygt9u26Bn?with_counts=true", {
      signal: AbortSignal.timeout(3000),
    })
    if (res.ok) {
      const data = await res.json()
      cached = data.approximate_member_count ?? null
    }
  } catch {}
  return cached
}
