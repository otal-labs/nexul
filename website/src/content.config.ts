import { defineCollection } from 'astro:content';
import { glob } from 'astro/loaders';
import { z } from 'astro/zod';

// Ids drop the extension and a trailing /index, so docs/guide/index.md is docs/guide and its URL is /docs/guide/.
const entryId = ({ entry }: { entry: string }) => entry.replace(/\.md$/, '').replace(/\/?index$/, '') || 'index';

export const collections = {
	docs: defineCollection({
		loader: glob({ pattern: '**/*.md', base: './src/content/docs', generateId: entryId }),
		schema: z.object({
			title: z.string(),
			description: z.string(),
			sidebar: z.object({ label: z.string().optional(), order: z.number().optional() }).optional(),
		}),
	}),
};
