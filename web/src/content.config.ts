import { defineCollection, z } from "astro:content"
import { glob } from "astro/loaders"

const articles = defineCollection({
  loader: glob({ pattern: "**/*.{md,mdx}", base: "./src/content/articles" }),
  schema: z.object({
    title: z.string(),
    description: z.string(),
    tags: z.array(z.string()).optional().default([]),
    order: z.number().optional().default(99),
    authors: z.array(z.string()).optional().default([]),
    date: z.coerce.date().optional(),
    image: z.string().optional(),
    featured: z.boolean().optional().default(false),
    featuredLabel: z.string().optional(),
    published: z.boolean().optional().default(true),
  }),
})

export const collections = { articles }
