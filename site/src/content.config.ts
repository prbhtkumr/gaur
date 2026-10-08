import { defineCollection } from 'astro:content';
import { glob } from 'astro/loaders';
import { z } from 'astro/zod';

// Content Layer API (required since Astro 6 removed the legacy collections
// that lived in src/content/config.ts with `type: 'content'`). The glob loader
// yields the same path-based IDs the old API exposed as `slug` for these flat
// files (installation, usage, configuration, internals).
const docs = defineCollection({
  loader: glob({ pattern: '**/[^_]*.md', base: './src/content/docs' }),
  schema: z.object({
    title: z.string(),
    description: z.string().optional(),
  }),
});

export const collections = { docs };
