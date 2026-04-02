/**
 * Astro auto-prefixes hrefs in .astro files but NOT inside .tsx components.
 * Use url() everywhere you build internal links in React components.
 */
export const BASE = import.meta.env.BASE_URL.replace(/\/$/, "")
export const url = (path: string) => `${BASE}${path}`
