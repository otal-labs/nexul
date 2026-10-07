import type { APIRoute } from 'astro';
import { createSearchAPI } from 'fumadocs-core/search/server';
import { structure } from 'fumadocs-core/mdx-plugins';
import { docEntries } from '../../lib/docs-content';
import { docUrl } from '../../lib/docs-tree';

export const GET: APIRoute = async () => {
	const indexes = (await docEntries()).map((entry) => ({
		id: entry.id,
		url: docUrl(entry.id),
		title: entry.data.title,
		description: entry.data.description,
		structuredData: structure(entry.body ?? ''),
	}));
	return createSearchAPI('advanced', { indexes }).staticGET();
};
