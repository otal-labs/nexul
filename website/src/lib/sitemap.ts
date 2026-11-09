const escapeXml = (text: string) => text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

// Each path becomes the absolute, trailing-slash URL its page names as canonical.
export const sitemapXml = (site: URL, paths: string[]) => {
	const urls = paths.map((path) => `\t<url><loc>${escapeXml(new URL(path.replace(/\/?$/, '/'), site).href)}</loc></url>`);
	return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls.join('\n')}\n</urlset>\n`;
};
