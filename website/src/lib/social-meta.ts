export const previewImage = {
	path: '/social-preview.png',
	width: 1200,
	height: 630,
	alt: 'Nexul: from docs to deploy, on your servers.',
};

interface Page {
	title: string;
	description: string;
	url: URL;
	type: 'website' | 'article';
}

export type MetaTag = { property: string; content: string } | { name: string; content: string };

// X reads only the twitter: names; Slack, Discord, LinkedIn and iMessage read the og: properties.
export function socialMeta({ title, description, url, type }: Page): MetaTag[] {
	const image = new URL(previewImage.path, url).href;
	return [
		{ property: 'og:site_name', content: 'Nexul' },
		{ property: 'og:type', content: type },
		{ property: 'og:url', content: url.href },
		{ property: 'og:title', content: title },
		{ property: 'og:description', content: description },
		{ property: 'og:image', content: image },
		{ property: 'og:image:type', content: 'image/png' },
		{ property: 'og:image:width', content: String(previewImage.width) },
		{ property: 'og:image:height', content: String(previewImage.height) },
		{ property: 'og:image:alt', content: previewImage.alt },
		{ name: 'twitter:card', content: 'summary_large_image' },
		{ name: 'twitter:title', content: title },
		{ name: 'twitter:description', content: description },
		{ name: 'twitter:image', content: image },
		{ name: 'twitter:image:alt', content: previewImage.alt },
	];
}
