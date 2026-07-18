// wbm-relay is a separate, private backend (not in this repo) that
// proxies/caches a reverse-engineered WWM gallery API. This file only
// talks to its small public API — no credentials or upstream details
// live here. Override PUBLIC_WBM_RELAY_URL in web/.env for local dev
// against a local `task dev` relay instance.
export const WBM_RELAY_URL = import.meta.env.PUBLIC_WBM_RELAY_URL || "http://localhost:3000"

export interface GalleryPlan {
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
}
