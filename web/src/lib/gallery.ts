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
// exist on PlanDetail (GET /api/plan?id=, see planDetailUrl).
export interface GalleryPlan {
  plan_id: string
  art_code: string
  share_id: string
  picture_url: string
  // build_num is a download/apply count, not a piece count despite the
  // name — see wbm-relay's CLAUDE.md "Component list" section. Use
  // components_count for the actual piece total.
  build_num: number
  like_num: number
  heat_val: number
  day_hot: number
  week_hot: number
  private: number
  upload_ts: number
  category_tag: number
  // The real total placed-piece count (NetEase's own
  // extra.components_count field) — 0/absent if upstream didn't include
  // it for this item.
  components_count?: number
  // author_number_id is also what designerUrl()/GET /api/designer takes
  // — the relay resolves whatever internal id it needs server-side, so
  // that's never exposed here. See wbm-relay's CLAUDE.md "Designer
  // profiles" section.
  author_number_id?: string
  author_name?: string
  // Real in-game character portrait, resolved server-side from a
  // third-party (yysls.cn) catalog keyed by NetEase's own head.role_icon
  // field — see wbm-relay's CLAUDE.md/avatar.go. Empty/absent if
  // resolution failed, same non-fatal pattern as author_name.
  author_avatar_url?: string
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
  // Resolved server-side (one extra get_players_info call) since,
  // unlike a gallery card, a direct link to this plan (e.g. via
  // planDetailUrl with a SHARE code) has no other source for this.
  author_number_id?: string
  author_name?: string
  author_avatar_url?: string
  // See GalleryPlan.build_num — a download/apply count, not a piece
  // count. components_count below is the real piece total.
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
  // Sum of every value in components — the real total piece count.
  components_count?: number
  // Only set on "industry"/settlement-type plans (guild bases etc) —
  // undefined otherwise.
  industry_level?: number
  prosperity?: number
}

// Mirrors wbm-relay's pkg/relay.Comment — one entry in a plan's
// guestbook-style comment thread (GET /api/comments?plan_id=, see
// commentsUrl). Confirmed live 2026-07-20 via a real wbm-tool capture of
// the in-game comment panel — author_number_id/author_name are resolved
// server-side the same way as PlanDetail's.
export interface Comment {
  comment_id: string
  msg: string
  ts: number
  author_number_id?: string
  author_name?: string
  author_avatar_url?: string
}

// id accepts a bare plan_id, an ART code, or a SHARE code — wbm-relay
// resolves whichever it got server-side (see its CLAUDE.md's
// "Shareable plan links" section). plan_id specifically can contain "/"
// and "+" (e.g. "aluKZe6M5w8b/fFP") — never safe as a raw path segment
// even URL-encoded, so always use the query form regardless of which
// code type this is.
export function planDetailUrl(id: string): string {
  return `${WBM_RELAY_URL}/api/plan?id=${encodeURIComponent(id)}`
}

// planID must be a bare plan_id (e.g. PlanDetail.plan_id) — unlike
// planDetailUrl, the relay's /api/comments doesn't resolve ART/SHARE
// codes, since every caller already has a resolved plan_id on hand by
// the time it wants comments (it only fetches these after a plan detail
// has already loaded).
export function commentsUrl(planID: string): string {
  return `${WBM_RELAY_URL}/api/comments?plan_id=${encodeURIComponent(planID)}`
}

// numberID is the public account number (author_number_id on a
// GalleryPlan) — the relay resolves whatever internal id it needs
// server-side, so nothing else has to travel through this URL.
export function designerUrl(numberID: string): string {
  return `${WBM_RELAY_URL}/api/designer?id=${encodeURIComponent(numberID)}`
}

// nickname must match exactly — confirmed live 2026-07-20, no known
// fuzzy/substring matching on this endpoint. See wbm-relay's CLAUDE.md
// "Shareable plan links"-adjacent designer-profile section.
export function designerByNameUrl(nickname: string): string {
  return `${WBM_RELAY_URL}/api/designer?name=${encodeURIComponent(nickname)}`
}

// Confirmed working for ART codes (e.g. "ARTakLUQfFVevW1Xl1A") and SHARE
// codes (e.g. "SHARE5f223181ad510813") — see wbm-relay's
// wbm-tool/gallery-api.md. Free-text title search isn't confirmed
// upstream, so a plain name may return no results.
export function searchGalleryUrl(q: string): string {
  return `${WBM_RELAY_URL}/api/search?q=${encodeURIComponent(q)}`
}

// ART = "ART" + base64 of a 12-byte value (16 base64 chars, no padding).
// SHARE = "SHARE" + 8 raw hex bytes (16 hex chars). See wbm-relay's
// wwm-presets/architecture.md "What the codes are". Free-text isn't a
// confirmed upstream search mode, so reject anything that isn't one of
// these two shapes before hitting the relay.
const ART_CODE_RE = /^ART[A-Za-z0-9+/]{16}$/
const SHARE_CODE_RE = /^SHARE[0-9a-f]{16}$/i

export function isValidGalleryCode(q: string): boolean {
  return ART_CODE_RE.test(q) || SHARE_CODE_RE.test(q)
}

// private=1 reliably means "Only Visible To Me" — private=0 covers
// "Public," "Cannot Apply," AND "Friends Can Apply" alike, genuinely
// indistinguishable in NetEase's own data (see cmd/diagram-lookup's
// visibility() in the main repo, and gallery-api.md/architecture.md).
export function isPrivate(private_: number): boolean {
  return private_ === 1
}

// Richer 3-way label, only available where has_friends_whitelist is
// known (PlanDetail, i.e. the plan detail view — not grid thumbnails,
// which only ever have the coarser `private` flag). "Friends Only" isn't
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
  avatar_url?: string
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
