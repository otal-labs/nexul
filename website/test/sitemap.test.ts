import { expect, test } from 'bun:test';
import { sitemapXml } from '../src/lib/sitemap';

test('every path is listed once as the absolute trailing-slash URL its canonical names', () => {
	const xml = sitemapXml(new URL('https://nexul.io'), ['/', '/roadmap', '/docs/guide/install/']);
	const locs = [...xml.matchAll(/<loc>([^<]+)<\/loc>/g)].map((match) => match[1]);

	expect(xml.startsWith('<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">')).toBe(true);
	expect(locs).toEqual(['https://nexul.io/', 'https://nexul.io/roadmap/', 'https://nexul.io/docs/guide/install/']);
});
