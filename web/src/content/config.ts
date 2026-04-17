import { defineCollection, z } from "astro:content"

const articles = defineCollection({
  type: "content",
  schema: z.object({
    title: z.string(),
    description: z.string(),
    tags: z.array(z.string()).optional().default([]),
    order: z.number().optional().default(99),
    authors: z.array(z.string()).optional().default([]),
    date: z.coerce.date().optional(),
    featured: z.boolean().optional().default(false),
    featuredLabel: z.string().optional(),
  }),
})

export const collections = { articles }
