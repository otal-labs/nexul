import type { APIRoute } from 'astro';
import { docEntries } from '../lib/docs-content';
import { sitemapXml } from '../lib/sitemap';

export const GET: APIRoute = async ({ site }) => {
	const docs = (await docEntries()).map((entry) => `/${entry.id}`);
	return new Response(sitemapXml(site!, ['/', '/roadmap', '/changelog', ...docs]), {
		headers: { 'Content-Type': 'application/xml; charset=utf-8' },
	});
};
