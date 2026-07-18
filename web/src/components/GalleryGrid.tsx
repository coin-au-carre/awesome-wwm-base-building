import * as React from "react"
import { useEffect, useState } from "react"
import { Card, CardHeader, CardTitle, CardDescription, CardFooter } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { HammerIcon, HeartIcon, FireIcon } from "@phosphor-icons/react"
import { WBM_RELAY_URL, type GalleryPlan } from "@/lib/gallery"

export function GalleryGrid() {
  const [plans, setPlans] = useState<GalleryPlan[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch(`${WBM_RELAY_URL}/api/gallery`)
      .then((res) => {
        if (!res.ok) throw new Error(`relay returned ${res.status}`)
        return res.json()
      })
      .then((data) => setPlans(data.plans ?? []))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
  }, [])

  if (error) {
    return (
      <p className="text-sm text-muted-foreground">
        Gallery temporarily unavailable ({error}). Try again shortly.
      </p>
    )
  }

  if (!plans) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-64 rounded-xl" />
        ))}
      </div>
    )
  }

  if (plans.length === 0) {
    return <p className="text-sm text-muted-foreground">No builds in the gallery yet.</p>
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {plans.map((plan) => (
        <Card key={plan.plan_id} className="overflow-hidden pt-0">
          {plan.picture_url && (
            <img
              src={plan.picture_url}
              alt={plan.title || plan.art_code}
              className="w-full aspect-video object-cover"
              loading="lazy"
            />
          )}
          <CardHeader>
            <CardTitle className="text-base">{plan.title || plan.art_code}</CardTitle>
            {plan.description && <CardDescription>{plan.description}</CardDescription>}
          </CardHeader>
          <CardFooter className="flex items-center gap-3 text-xs text-muted-foreground">
            <span className="flex items-center gap-1">
              <HammerIcon weight="duotone" className="size-3.5" /> {plan.build_num}
            </span>
            <span className="flex items-center gap-1">
              <HeartIcon weight="duotone" className="size-3.5" /> {plan.like_num}
            </span>
            <span className="flex items-center gap-1">
              <FireIcon weight="duotone" className="size-3.5" /> {plan.heat_val}
            </span>
          </CardFooter>
        </Card>
      ))}
    </div>
  )
}
