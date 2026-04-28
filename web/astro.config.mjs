// @ts-check

import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from "astro/config"
import react from "@astrojs/react"
import sitemap from "@astrojs/sitemap"

// https://astro.build/config
export default defineConfig({
  site: "https://www.wherebuildersmeet.com",
  output: "static",
  vite: {
    plugins: [tailwindcss()],
  },
  integrations: [
    react(),
    sitemap({
      serialize(item) {
        const url = item.url
        if (/\/(guilds|solos)\/[^/]+\/?$/.test(url)) {
          item.priority = 0.4
          item.changefreq = "monthly"
          return item
        }
        if (/\/tutorials\/[^/]+\/?$/.test(url)) {
          item.priority = 0.8
          item.changefreq = "monthly"
          return item
        }
        // Homepage
        if (/wherebuildersmeet\.com\/?$/.test(url)) {
          item.priority = 1.0
          item.changefreq = "daily"
          return item
        }
        // All other top-level pages (solo, tutorials index, events, media, catalog, contribute, how-it-works)
        item.priority = 0.9
        item.changefreq = "weekly"
        return item
      },
    }),
  ],
})
