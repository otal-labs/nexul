import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import { sidebar } from './src/lib/sidebar';

export default defineConfig({
	site: 'https://nexul.io',
	integrations: [
		starlight({
			title: 'Nexul',
			disable404Route: true,
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
			sidebar,
			editLink: { baseUrl: 'https://github.com/otal-labs/nexul/edit/master/website/' },
			components: {
				PageTitle: './src/components/docs/PageTitle.astro',
				SocialIcons: './src/components/docs/SocialIcons.astro',
			},
		}),
	],
});
