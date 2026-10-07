import type { APIRoute, GetStaticPaths } from 'astro';
import type { CollectionEntry } from 'astro:content';
import { docEntries, pageMarkdown } from '../lib/docs-content';

export const getStaticPaths = (async () =>
	(await docEntries()).map((entry) => ({ params: { slug: entry.id }, props: { entry } }))) satisfies GetStaticPaths;

export const GET: APIRoute = ({ props }) =>
	new Response(pageMarkdown((props as { entry: CollectionEntry<'docs'> }).entry), {
		headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
	});
