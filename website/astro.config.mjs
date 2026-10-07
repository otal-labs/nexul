import { defineConfig } from 'astro/config';
import react from '@astrojs/react';
import tailwindcss from '@tailwindcss/vite';
import { unified } from '@astrojs/markdown-remark';
import { rehypeCode } from 'fumadocs-core/mdx-plugins';
import { monoDark, monoLight } from './src/lib/shiki-mono';

export default defineConfig({
	site: 'https://nexul.io',
	markdown: {
		processor: unified({
			syntaxHighlight: false,
			rehypePlugins: [[rehypeCode, { icon: false, themes: { light: monoLight, dark: monoDark } }]],
		}),
	},
	integrations: [react()],
	vite: { plugins: [tailwindcss()] },
});
