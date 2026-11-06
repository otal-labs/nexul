import type { APIRoute } from 'astro';
import { structure } from 'fumadocs-core/mdx-plugins';
import { docEntries } from '../../lib/docs-content';
import { docUrl } from '../../lib/docs-tree';

// Names the section behind each anchor URL, and what each page covers, for search rows that have only a URL.
export const GET: APIRoute = async () => {
	const headings: Record<string, string> = {};
	const descriptions: Record<string, string> = {};
	for (const entry of await docEntries()) {
		const url = docUrl(entry.id);
		descriptions[url] = entry.data.description;
		for (const heading of structure(entry.body ?? '').headings) headings[`${url}#${heading.id}`] = heading.content;
	}
	return new Response(JSON.stringify({ headings, descriptions }), { headers: { 'Content-Type': 'application/json' } });
};
