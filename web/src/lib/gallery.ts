// wbm-relay is a separate, private backend (not in this repo) that
// proxies/caches a reverse-engineered WWM gallery API. This file only
// talks to its small public API — no credentials or upstream details
// live here. Override PUBLIC_WBM_RELAY_URL in web/.env for local dev
// against a local `task dev` relay instance.
export const WBM_RELAY_URL = import.meta.env.PUBLIC_WBM_RELAY_URL || "http://localhost:3000"

// Mirrors wbm-relay's pkg/relay.PlanBrief — a gallery-listing item, not
// the full plan detail. No title/description/previews: those only
// exist on GET /api/plan/:id, which isn't wired into this page.
export interface GalleryPlan {
  plan_id: string
  art_code: string
  share_id: string
  picture_url: string
  build_num: number
  like_num: number
  heat_val: number
  day_hot: number
  week_hot: number
  private: number
  upload_ts: number
  category_tag: number
  author_number_id?: string
  author_name?: string
}

export interface GalleryPage {
  plans: GalleryPlan[]
  next_start: number
}

// Mirrors wbm-relay's pkg/relay.PlanDetail — the full single-plan
// response from GET /api/plan?id=, fetched on demand (one call) when a
// visitor opens a specific build, not as part of the gallery listing.
export interface PlanDetail {
  plan_id: string
  art_code: string
  share_id: string
  title: string
  description: string
  picture_url: string
  previews: string[] | null
  build_num: number
  like_num: number
  heat_val: number
  private: number
  upload_ts: number
}

// plan_id can contain "/" and "+" (e.g. "aluKZe6M5w8b/fFP") — never
// safe as a raw path segment even URL-encoded. Always use the query
// form. See wbm-relay's CLAUDE.md.
export function planDetailUrl(planID: string): string {
  return `${WBM_RELAY_URL}/api/plan?id=${encodeURIComponent(planID)}`
}

// 1464320 -> "1464K". Matches how the in-game UI abbreviates large
// heat/like counts.
export function formatCount(n: number): string {
  if (n >= 1000) {
    return `${Math.floor(n / 1000)}K`
  }
  return String(n)
}

// Mirrors wbm-relay's relay.SortToRecType keys. "recommended" currently
// returns identical ordering to "trending_alltime" on this community's
// dataset (see wbm-relay's gallery-api.md) — included anyway for parity
// with the in-game tab, may diverge as the catalog grows.
export const SORT_OPTIONS = [
  { value: "recommended", label: "Recommended" },
  { value: "latest", label: "Latest" },
  { value: "trending_today", label: "Trending Today" },
  { value: "trending_weekly", label: "Trending Weekly" },
  { value: "trending_alltime", label: "All-time" },
] as const

export const DEFAULT_SORT: (typeof SORT_OPTIONS)[number]["value"] = "recommended"

// All 12 in-game category tag ids, confirmed 2026-07-18 (see
// wbm-relay's wbm-tool/gallery-api.md). "Small World" (950) and
// "Painted Boat Diagram" (1211) genuinely return 0 items right now —
// confirmed via a real fresh fetch, not a stale-cache artifact — kept
// in the list since that can change as more builds get uploaded.
export const CATEGORY_OPTIONS = [
  { value: 9, label: "All" },
  { value: 900, label: "Cloudrest Passage" },
  { value: 901, label: "House" },
  { value: 902, label: "Blissful Retreat" },
  { value: 903, label: "Porcelain Kiln" },
  { value: 904, label: "Aromas Brewery" },
  { value: 905, label: "Crane Retreat" },
  { value: 906, label: "Residence" },
  { value: 950, label: "Small World" },
  { value: 1100, label: "Guild Base" },
  { value: 1200, label: "Small Diagram" },
  { value: 1211, label: "Painted Boat Diagram" },
] as const

// Looks up a category's display label by its tag id — falls back to
// null when the id isn't one of the mapped CATEGORY_OPTIONS (e.g. a
// category never captured yet).
export function categoryLabel(tag: number): string | null {
  return CATEGORY_OPTIONS.find((opt) => opt.value === tag)?.label ?? null
}
