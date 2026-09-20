import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
	integrations: [
		starlight({
			title: 'Nexul',
			favicon: '/favicon.svg',
			logo: {
				light: './src/assets/nexul-icon.svg',
				dark: './src/assets/nexul-icon-dark.svg',
				alt: 'Nexul',
			},
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/otal-labs/nexul' },
			],
			customCss: ['./src/styles/mono-console.css'],
			sidebar: [
				{
					label: 'Using Nexul',
					items: [{ autogenerate: { directory: 'docs/guide' } }],
				},
				{
					label: 'Contributing',
					items: [{ autogenerate: { directory: 'docs/contributing' } }],
				},
			],
		}),
	],
});
