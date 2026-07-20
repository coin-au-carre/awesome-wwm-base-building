import * as React from "react"
import { useEffect, useState } from "react"
import { Skeleton } from "@/components/ui/skeleton"
import { UserCircleIcon } from "@phosphor-icons/react"
import { PlanCard, CopyPill, ShareButton } from "@/components/GalleryGrid"
import { BackLink, GalleryLink } from "@/components/BackLink"
import { WBM_RELAY_URL, designerUrl, designerByNameUrl, type DesignerProfile } from "@/lib/gallery"

function StatTile({ value, label }: { value: number | string; label: string }) {
  return (
    <div className="text-center">
      <div className="font-heading text-2xl sm:text-3xl font-bold italic">{value}</div>
      <div className="text-xs text-muted-foreground">{label}</div>
    </div>
  )
}

export function BuilderProfile() {
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
    setQuery({ id: params.get("id"), name: params.get("name") })
  }, [])

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
      .then((data) => data && setProfile(data))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
  }, [query])

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
      <div className="flex flex-wrap items-center gap-6">
        <div className="flex items-center gap-3">
          <UserCircleIcon weight="fill" className="size-14 text-muted-foreground/50" />
          <h1 className="font-heading text-3xl font-bold leading-tight">{profile.nickname || profile.number_id}</h1>
        </div>
        <div className="flex items-center gap-6 ml-auto">
          <StatTile value={profile.follower_num} label="Fans" />
          <StatTile value={profile.like_num} label="Likes" />
          <StatTile value={profile.published_num} label="Published Works" />
        </div>
      </div>

      {profile.plans.length === 0 ? (
        <p className="text-sm text-muted-foreground">No published construction diagrams yet.</p>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
            <span>
              Showing all {profile.plans.length} published construction diagrams by {profile.nickname || profile.number_id}.
            </span>
            {profile.number_id && <CopyPill label="ID" value={profile.number_id} />}
          </div>
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
