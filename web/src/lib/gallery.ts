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

// Aggregate feed of every known WBM builder's public construction
// diagrams (see wbm-relay's relay.FetchWBMBuilderPlans and this repo's
// docs/builder-identity.md) — a purpose-built endpoint instead of
// paginating/filtering the general /api/gallery feed client-side.
// sort/tag are the same SORT_OPTIONS/CATEGORY_OPTIONS values as
// /api/gallery — the relay applies them locally to its own cached
// aggregate rather than querying upstream per combination.
// minComponents/maxComponents filter by total placed-piece count — only
// meaningful here (the WBM-only aggregate), not on the general gallery
// feed, since wbm-relay holds this whole list in memory to filter
// locally rather than sending an upstream request per range. Omit
// either bound to leave that side unbounded.
export function wbmGalleryUrl(
  sort: string,
  tag: number,
  start: number,
  limit: number,
  minComponents?: number,
  maxComponents?: number,
): string {
  const params = new URLSearchParams({ sort, tag: String(tag), start: String(start), limit: String(limit) })
  if (minComponents != null) params.set("min", String(minComponents))
  if (maxComponents != null) params.set("max", String(maxComponents))
  return `${WBM_RELAY_URL}/api/gallery/wbm?${params.toString()}`
}

// Every known WBM builder's avatar URL, keyed by number_id, in one
// response — one batched upstream call server-side (see wbm-relay's
// relay.FetchWBMBuilderAvatars) instead of one designerUrl request per
// builder just to show a whole roster's pictures (e.g. the builders
// directory).
export function wbmAvatarsUrl(): string {
  return `${WBM_RELAY_URL}/api/gallery/wbm/avatars`
}

// Mirrors wbm-relay's relay.BuilderStatus — refreshed far more often
// than avatars (see wbm-relay's wbmStatusCache, 1 min TTL vs. avatars'
// 10 min) since online status goes stale in minutes, not hours.
// oversea_tag/device_name/max_xiuwei_kungfu ride along on the same
// upstream call (no extra cost) so the directory can show/filter/sort a
// whole roster by these without a per-row designerUrl call.
export interface BuilderStatus {
  level: number
  is_online: boolean
  oversea_tag?: string
  device_name?: string
  max_xiuwei_kungfu?: number
}

// device_name is NetEase's raw internal platform string, not a
// player-facing name — "prospero" is Sony's PS5 platform codename
// (Xbox's equivalent, unseen so far, would presumably be "scarlett").
// Falls back to capitalizing whatever's given so an unmapped value
// (e.g. "windows"/"android"/"ios") still reads reasonably.
const DEVICE_LABELS: Record<string, string> = {
  prospero: "PS5",
  scarlett: "Xbox Series X|S",
}

export function deviceLabel(deviceName: string): string {
  return DEVICE_LABELS[deviceName.toLowerCase()] ?? (deviceName.charAt(0).toUpperCase() + deviceName.slice(1))
}

// Every known WBM builder's level/online status, keyed by number_id, in
// one response — same "resolve once for the whole roster" pattern as
// wbmAvatarsUrl, just on its own short-TTL cache server-side.
export function wbmStatusUrl(): string {
  return `${WBM_RELAY_URL}/api/gallery/wbm/status`
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

// Mirrors wbm-relay's pkg/relay.HomeSpace — a pointer to a homestead
// space (id/level/name), not its contents. See modPlayerUrl.
export interface HomeSpace {
  space_id: string
  space_no: number
  level: number
  name?: string
}

// Mirrors wbm-relay's pkg/relay.ModPlayerDetail — real PII (bio, linked
// third-party accounts) only ever returned by the mod-gated
// GET /api/mod/player endpoint, never by designerUrl()/DesignerProfile.
// Used exclusively by copyright-watch.astro's MonitorEntry, behind the
// MOD_SECRET key gate (see ModKeyGate.tsx). Never fetch this for a
// public-facing view.
export interface ModPlayerDetail {
  number_id: string
  nickname: string
  level: number
  oversea_tag?: string
  is_online: boolean
  discord_account_id?: string
  discord_global_name?: string
  steam_account_id?: string
  xbox_account_id?: string
  xbox_username?: string
  psn_user_name?: string
  bio?: string
  avatar_url?: string
  // Unix seconds (create_time/login_time/logout_time) / cumulative
  // playtime in seconds (online_time) — added 2026-07-22.
  create_time?: number
  login_time?: number
  logout_time?: number
  online_time?: number
  max_xiuwei_kungfu?: number
  device_name?: string
  home_spaces?: HomeSpace[]
  home_works?: HomeWork[]
  campaign_slogan?: string
  campaign_banner_url?: string
}

// Mirrors wbm-relay's pkg/relay.HomeWork — a showcased-work pointer
// (school.homepage_ly), possibly including builds not shown in the
// player's public gallery listing. work_type's exact meaning isn't
// decoded upstream yet — see wbm-relay's gallery-api.md.
export interface HomeWork {
  work_id: string
  work_type: number
}

// key is the shared mod secret (see ModKeyGate.tsx) — sent as a query
// param since this is a plain GET, checked server-side against
// MOD_SECRET (constant-time compare, wbm-relay's pkg/server/moddata.go).
// A wrong/missing key gets a 401, not partial data.
export function modPlayerUrl(numberID: string, key: string): string {
  return `${WBM_RELAY_URL}/api/mod/player?id=${encodeURIComponent(numberID)}&key=${encodeURIComponent(key)}`
}

// No payload, just a 200/401 — used only to validate a key before
// revealing copyright-watch.astro's page content at all, see
// ModKeyGate.tsx. Same MOD_SECRET check as modPlayerUrl, no player
// lookup involved.
export function modCheckUrl(key: string): string {
  return `${WBM_RELAY_URL}/api/mod/check?key=${encodeURIComponent(key)}`
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
  // Public since 2026-07-22 — see wbm-relay's designer.go doc comment.
  // max_xiuwei_kungfu is labeled "Martial Mastery" in this site's UI,
  // matching the in-game name for this stat more closely than the raw
  // upstream field name.
  level?: number
  oversea_tag?: string
  is_online: boolean
  device_name?: string
  max_xiuwei_kungfu?: number
  // Unix seconds — power "Logged in since" (online) / "Last seen"
  // (offline) on the builders directory/profile. Public since 2026-07-22.
  login_time?: number
  logout_time?: number
  bio?: string
  campaign_slogan?: string
  campaign_banner_url?: string
  home_works?: HomeWork[]
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

// Mirrors wbm-relay's relay.SortToRecType keys. "recommended" and
// "trending_today" are omitted from the UI (redundant with "All-time" on
// this community's small dataset, and too noisy a set of tabs) — the
// backend still supports both if that changes.
export const SORT_OPTIONS = [
  { value: "trending_alltime", label: "All-time" },
  { value: "latest", label: "Latest" },
  { value: "trending_weekly", label: "Trending Weekly" },
] as const

export const DEFAULT_SORT: (typeof SORT_OPTIONS)[number]["value"] = "trending_alltime"

// "components" only exists on wbm-relay's /api/gallery/wbm — that handler
// already holds the whole WBM aggregate in memory to re-sort locally, so
// it can offer a sort key that isn't a real NetEase rec_type. The general
// /api/gallery feed forwards SORT_OPTIONS' value straight upstream as
// rec_type and has no such option — see wbm-relay's wbmgallery.go.
export const WBM_SORT_OPTIONS = [...SORT_OPTIONS, { value: "components", label: "Most Components" }] as const

// In-game category tag ids, confirmed 2026-07-18 (see wbm-relay's
// wbm-tool/gallery-api.md). "Small World" (950) and "Painted Boat
// Diagram" (1211) are omitted for now — they genuinely return 0 items,
// confirmed via a real fresh fetch, not a stale-cache artifact. Re-add
// if builds ever start showing up in those categories.
export const CATEGORY_OPTIONS = [
  { value: 9, label: "All" },
  { value: 900, label: "Cloudrest Passage" },
  { value: 901, label: "House" },
  { value: 902, label: "Blissful Retreat" },
  { value: 903, label: "Porcelain Kiln" },
  { value: 904, label: "Aromas Brewery" },
  { value: 905, label: "Crane Retreat" },
  { value: 906, label: "Residence" },
  { value: 1100, label: "Guild Base" },
  { value: 1200, label: "Small Diagram" },
] as const

// Looks up a category's display label by its tag id — falls back to
// null when the id isn't one of the mapped CATEGORY_OPTIONS (e.g. a
// category never captured yet).
export function categoryLabel(tag: number): string | null {
  return CATEGORY_OPTIONS.find((opt) => opt.value === tag)?.label ?? null
}
