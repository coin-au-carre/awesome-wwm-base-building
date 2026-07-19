// wbm-relay is a separate, private backend (not in this repo) that
// proxies/caches a reverse-engineered WWM gallery API. This file only
// talks to its small public API — no credentials or upstream details
// live here. Override PUBLIC_WBM_RELAY_URL in web/.env for local dev
// against a local `task dev` relay instance.
//
// wbm-relay has no public deployment yet, so the localhost fallback
// only applies in dev — baking it into the production build made every
// visitor's browser try to fetch a private-network address, which
// browsers flag with a Local Network Access permission prompt. Empty
// in prod until PUBLIC_WBM_RELAY_URL is wired up as a real deploy secret.
export const WBM_RELAY_URL = import.meta.env.PUBLIC_WBM_RELAY_URL || (import.meta.env.DEV ? "http://localhost:3000" : "")

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
  // author_number_id is also what designerUrl()/GET /api/designer takes
  // — the relay resolves whatever internal id it needs server-side, so
  // that's never exposed here. See wbm-relay's CLAUDE.md "Designer
  // profiles" section.
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
  // Only ever populated here (get_face_plan_data) — never on
  // GalleryPlan/DesignerPlan, which come from endpoints that don't
  // return this field at all.
  has_friends_whitelist: boolean
  // Bill of materials: {type_code: count}, all spatial data stripped.
  // Keys are raw internal item-type codes (e.g. "803905") — no
  // human-readable name mapping exists yet. See wbm-relay's CLAUDE.md
  // "Bill of materials" section. Undefined when the plan has none.
  components?: Record<string, number>
  // Only set on "industry"/settlement-type plans (guild bases etc) —
  // undefined otherwise.
  industry_level?: number
  prosperity?: number
}

// plan_id can contain "/" and "+" (e.g. "aluKZe6M5w8b/fFP") — never
// safe as a raw path segment even URL-encoded. Always use the query
// form. See wbm-relay's CLAUDE.md.
export function planDetailUrl(planID: string): string {
  return `${WBM_RELAY_URL}/api/plan?id=${encodeURIComponent(planID)}`
}

// numberID is the public account number (author_number_id on a
// GalleryPlan) — the relay resolves whatever internal id it needs
// server-side, so nothing else has to travel through this URL.
export function designerUrl(numberID: string): string {
  return `${WBM_RELAY_URL}/api/designer?id=${encodeURIComponent(numberID)}`
}

// Confirmed working for ART codes (e.g. "ARTakLUQfFVevW1Xl1A") and SHARE
// codes (e.g. "SHAREeaea710c24cbc453") — see wbm-relay's
// wbm-tool/gallery-api.md. Free-text title search isn't confirmed
// upstream, so a plain name may return no results.
export function searchGalleryUrl(q: string): string {
  return `${WBM_RELAY_URL}/api/search?q=${encodeURIComponent(q)}`
}

// private=1 reliably means "Only Visible To Me" — private=0 covers
// "Public," "Cannot Apply," AND "Friends Can Apply" alike, genuinely
// indistinguishable in NetEase's own data (see cmd/diagram-lookup's
// visibility() in the main repo, and gallery-api.md/architecture.md).
export function isPrivate(private_: number): boolean {
  return private_ === 1
}

// Richer 3-way label, only available where has_friends_whitelist is
// known (PlanDetail, i.e. the detail modal — not grid thumbnails, which
// only ever have the coarser `private` flag). "Friends Only" isn't
// independently confirmed against a real Friends-Can-Apply diagram yet
// — see wbm-relay's CLAUDE.md "Sharing-permission ambiguity".
export function planVisibility(private_: number, hasFriendsWhitelist: boolean): "private" | "friends" | "public" {
  if (private_ === 1) return "private"
  if (hasFriendsWhitelist) return "friends"
  return "public"
}

// Mirrors wbm-relay's pkg/relay.DesignerProfile. Lighter stats than the
// in-game designer profile UI (no "Total Popularity"/"Following") — that
// data has no known equivalent on NetEase's live API. See wbm-relay's
// CLAUDE.md "Designer profiles" section.
export interface DesignerProfile {
  // number_id is the same public account number passed to designerUrl()
  // — the internal id wbm-relay actually queries with is never exposed.
  number_id: string
  nickname: string
  follower_num: number
  like_num: number
  published_num: number
  plans: DesignerPlan[]
}

// wbm-relay's DesignerProfile.plans is now []*PlanBrief directly (the
// live host's plan-brief-batch endpoint returns build_num/like_num per
// plan, unlike the old wwmpresets.com proxy this replaced) — same shape
// as GalleryPlan, no separate lighter type needed anymore.
export type DesignerPlan = GalleryPlan

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
