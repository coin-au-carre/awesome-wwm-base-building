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
      filter: (page) => !page.includes("/admin/") && !/\/media\/[^/]+\/?$/.test(page),
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
