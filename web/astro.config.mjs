// @ts-check

import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from "astro/config"
import react from "@astrojs/react"
import sitemap, { ChangeFreqEnum } from "@astrojs/sitemap"
import remarkBreaks from "remark-breaks"
import { remarkDiscordSubheading } from "./src/lib/remark-discord-subheading.mjs"
import rehypeAutolinkHeadings from "rehype-autolink-headings"
import rehypeSlug from "rehype-slug"

// https://astro.build/config
export default defineConfig({
  site: "https://www.wherebuildersmeet.com",
  output: "static",
  build: {
    inlineStylesheets: "always",
  },
  prefetch: {
    prefetchAll: true,
    defaultStrategy: "hover",
  },
  vite: {
    plugins: [tailwindcss()],
  },
  markdown: {
    remarkPlugins: [remarkBreaks, remarkDiscordSubheading],
    rehypePlugins: [
      rehypeSlug,
      [rehypeAutolinkHeadings, { behavior: "wrap" }],
    ],
  },
  integrations: [
    react(),
    sitemap({
      // /gallery itself is now indexable (public, server-rendered browse
      // page) — /gallery/builder and /gallery/plan stay out of the
      // sitemap since they're query-string routes with no meaningful
      // content at their bare URL (no getStaticPaths, content only
      // exists once ?id=/?share= is present and fetched client-side).
      filter: (page) => !page.includes("/admin/") && !page.includes("/gallery/builder") && !page.includes("/gallery/plan") && !page.includes("/copyright-watch") && !/\/media\/[^/]+\/?$/.test(page),
      serialize(item) {
        const url = item.url
        if (/\/(guilds|solos)\/[^/]+\/?$/.test(url)) {
          item.priority = 0.6
          item.changefreq = ChangeFreqEnum.WEEKLY
          return item
        }
        if (/\/tutorials\/[^/]+\/?$/.test(url)) {
          item.priority = 0.7
          item.changefreq = ChangeFreqEnum.MONTHLY
          return item
        }
        // Homepage
        if (/wherebuildersmeet\.com\/?$/.test(url)) {
          item.priority = 1.0
          item.changefreq = ChangeFreqEnum.DAILY
          return item
        }
        // All other top-level pages (solo, tutorials index, events, media, catalog, contribute, how-it-works)
        item.priority = 0.9
        item.changefreq = ChangeFreqEnum.WEEKLY
        return item
      },
    }),
  ],
})
