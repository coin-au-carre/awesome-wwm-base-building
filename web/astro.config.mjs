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
  integrations: [react(), sitemap({ filter: (page) => !page.includes("/tutorials/") })],
})
