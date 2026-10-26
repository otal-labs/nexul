import { expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { previewImage, socialMeta } from '../src/lib/social-meta';

const tags = socialMeta({ title: 'Roadmap · Nexul', description: 'What is being built.', url: new URL('https://nexul.io/roadmap/'), type: 'website', image: '/_astro/social-preview.Ab12Cd34.png' });
const content = (key: string) => tags.find((tag) => ('property' in tag ? tag.property : tag.name) === key)?.content;

test('a page carries the tags X and the other link unfurlers build a large card from', () => {
	expect(content('twitter:card')).toBe('summary_large_image');
	expect(content('og:url')).toBe('https://nexul.io/roadmap/');
	expect(content('og:site_name')).toBe('Nexul');
	expect(content('og:image')).toBe('https://nexul.io/_astro/social-preview.Ab12Cd34.png');
	expect(content('twitter:image')).toBe(content('og:image'));
	expect(content('twitter:title')).toBe('Roadmap · Nexul');
	expect(content('twitter:description')).toBe('What is being built.');
});

test('the preview image is a PNG at the size the tags declare and under the 5MB X accepts', () => {
	const png = readFileSync(new URL('../src/assets/social-preview.png', import.meta.url));

	expect(png.subarray(1, 4).toString()).toBe('PNG');
	expect([png.readUInt32BE(16), png.readUInt32BE(20)]).toEqual([previewImage.width, previewImage.height]);
	expect(png.length).toBeLessThan(5 * 1024 * 1024);
});
