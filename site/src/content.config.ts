// The docs collection: the pages cmd/docgen -site writes from docs/ into
// src/content/docs, with the front matter it adds (decision 0022).
import { defineCollection, z } from 'astro:content';
import { docsLoader } from '@astrojs/starlight/loaders';
import { docsSchema } from '@astrojs/starlight/schema';

export const collections = {
	docs: defineCollection({
		loader: docsLoader(),
		schema: docsSchema({
			extend: z.object({
				// The page's file in the repository, for "View the source".
				srcPath: z.string().optional(),
				// The page is written by cmd/docgen from the code.
				generated: z.boolean().optional(),
				// The landing page, which draws its own content.
				landing: z.boolean().optional(),
			}),
		}),
	}),
};
