import { defineConfig } from 'astro/config';
import react from '@astrojs/react';
import tailwindcss from '@tailwindcss/vite';
import { unified } from '@astrojs/markdown-remark';
import { rehypeCode } from 'fumadocs-core/mdx-plugins';
import { codeTheme } from './src/lib/shiki-mono';
import { rehypeCodeCopy } from './src/lib/rehype-code-copy';

export default defineConfig({
	site: 'https://nexul.io',
	markdown: {
		syntaxHighlight: false,
		processor: unified({
			syntaxHighlight: false,
			rehypePlugins: [[rehypeCode, { icon: false, themes: { light: codeTheme, dark: codeTheme } }], rehypeCodeCopy],
		}),
	},
	integrations: [react()],
	vite: { plugins: [tailwindcss()] },
});
