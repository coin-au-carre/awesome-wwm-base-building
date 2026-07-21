import * as React from "react"
import { useEffect, useState } from "react"
import { Skeleton } from "@/components/ui/skeleton"
import { PlanCard, CopyPill, ShareButton, Avatar } from "@/components/GalleryGrid"
import { BackLink, GalleryLink } from "@/components/BackLink"
import { WBM_RELAY_URL, designerUrl, designerByNameUrl, type DesignerProfile } from "@/lib/gallery"
import { url } from "@/lib/url"

export function StatTile({ value, label }: { value: number | string; label: string }) {
  return (
    <div className="text-center">
      <div className="font-heading text-2xl sm:text-3xl font-bold italic">{value}</div>
      <div className="text-xs text-muted-foreground">{label}</div>
    </div>
  )
}

// wbmSlugs maps a NetEase author_number_id to their WBM canonicalSlug —
// see data/builder_identities.json, computed server-side in
// gallery/builder.astro. A WBM-linked builder gets redirected to their
// fuller /builders/<slug> page instead of duplicating this profile across
// two URLs — see docs/builder-identity.md's "merging the two builder-
// profile systems". Builders with no WBM link (most gallery visitors)
// just render normally here, untouched.
export function BuilderProfile({ wbmSlugs = {} }: { wbmSlugs?: Record<string, string> }) {
  // undefined = not read from the URL yet (initial render), null = read
  // and genuinely absent. Only one of id/name is expected to be set.
  const [query, setQuery] = useState<{ id: string | null; name: string | null } | undefined>(undefined)
  const [profile, setProfile] = useState<DesignerProfile | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [notFound, setNotFound] = useState(false)

  // Query-string route (?id=<number_id> or ?name=<exact nickname>), not
  // a dynamic [id] path segment — the site builds statically with no
  // build-time list of designer ids. id is the public account number;
  // wbm-relay resolves whatever internal id it needs server-side (see
  // gallery.ts's designerUrl/designerByNameUrl). See gallery/builder.astro.
  useEffect(() => {
    const params = new URLSearchParams(location.search)
    const id = params.get("id")
    // Known WBM builder — redirect immediately, before ever fetching the
    // relay, so this page never renders anything for them.
    if (id && wbmSlugs[id]) {
      location.replace(url(`/builders/${wbmSlugs[id]}`))
      return
    }
    setQuery({ id, name: params.get("name") })
  }, [wbmSlugs])

  useEffect(() => {
    if (!query || (!query.id && !query.name)) return
    if (!WBM_RELAY_URL) {
      setError("not deployed yet")
      return
    }
    setError(null)
    setNotFound(false)
    const fetchUrl = query.id ? designerUrl(query.id) : designerByNameUrl(query.name!)
    fetch(fetchUrl)
      .then((res) => {
        if (res.status === 404) {
          setNotFound(true)
          return null
        }
        if (!res.ok) throw new Error(`relay returned ${res.status}`)
        return res.json()
      })
      .then((data) => {
        if (!data) { return }
        // Looked up by name — the id wasn't known until now. Redirect
        // the same way as the id-based path above.
        if (wbmSlugs[data.number_id]) {
          location.replace(url(`/builders/${wbmSlugs[data.number_id]}`))
          return
        }
        setProfile(data)
      })
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
  }, [query, wbmSlugs])

  if (query === undefined) return null

  if (!query.id && !query.name) {
    return <p className="text-sm text-muted-foreground">No builder specified.</p>
  }

  if (notFound) {
    return (
      <div className="space-y-4">
        <div className="flex gap-2">
          <BackLink />
          <GalleryLink />
        </div>
        <p className="text-sm text-muted-foreground">
          No results for "{query.id ?? query.name}". Double-check the ID or nickname — nicknames must match exactly.
        </p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-4">
        <div className="flex gap-2">
          <BackLink />
          <GalleryLink />
        </div>
        <p className="text-sm text-muted-foreground">Builder profile unavailable ({error}).</p>
      </div>
    )
  }

  if (!profile) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-20 w-full max-w-md rounded-xl" />
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="aspect-video rounded-xl" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <BackLink />
        <GalleryLink />
        <ShareButton label="Share profile" />
      </div>
      <div className="flex flex-wrap items-center gap-6 rounded-2xl ring-1 ring-border bg-card p-5 sm:p-6">
        <div className="flex items-center gap-5">
          <Avatar src={profile.avatar_url} className="flex size-14 sm:size-24" />
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="font-heading text-xl sm:text-3xl font-bold leading-tight">{profile.nickname || profile.number_id}</h1>
            {profile.number_id && <CopyPill label="ID" value={profile.number_id} />}
          </div>
        </div>
        <div className="flex items-center gap-6 sm:ml-auto">
          <StatTile value={profile.follower_num} label="Fans" />
          <StatTile value={profile.like_num} label="Likes" />
          <StatTile value={profile.published_num} label="Published Works" />
        </div>
      </div>

      {profile.plans.length === 0 ? (
        <p className="text-sm text-muted-foreground">No published construction diagrams yet.</p>
      ) : (
        <>
          <p className="text-sm text-muted-foreground">
            Showing all {profile.plans.length} published construction diagrams by {profile.nickname || profile.number_id}.
          </p>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
            {profile.plans.map((plan) => (
              <PlanCard key={plan.plan_id} plan={plan} showAuthor={false} />
            ))}
          </div>
        </>
      )}
    </div>
  )
}
