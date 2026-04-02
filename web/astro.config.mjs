// @ts-check

import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from "astro/config"
import react from "@astrojs/react"

// https://astro.build/config
export default defineConfig({
  site: "https://coin-au-carre.github.io",
  base: process.env.ASTRO_BASE ?? "/awesome-wwm-base-building",
  output: "static",
  vite: {
    plugins: [tailwindcss()],
  },
  integrations: [react()],
})
