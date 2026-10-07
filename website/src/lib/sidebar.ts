export const guide = (slug: string) => (slug === 'index' ? 'docs/guide' : `docs/guide/${slug}`);

export const guideGroups = [
	{ label: 'Get started', items: [guide('index'), guide('install'), guide('setup-wizard'), guide('github-app'), guide('upgrade')] },
	{ label: 'Deploy', items: [guide('runners'), guide('stacks-and-deploys'), guide('topology-and-dns'), guide('logs')] },
	{
		label: 'Work together',
		items: [
			guide('projects-and-repositories'),
			guide('people-and-access'),
			guide('docs-tickets-and-board'),
			guide('chat-and-voice'),
			guide('automations'),
		],
	},
	{
		label: 'Agents',
		items: [guide('computer-setup'), guide('paired-computers'), guide('plays'), guide('interview'), guide('memories'), guide('mcp-server')],
	},
	{ label: 'Reference', items: [guide('api-and-tokens'), guide('desktop-app'), guide('phone-app')] },
];

export const sidebar = [
	...guideGroups,
	{ label: 'Contributing', items: [{ autogenerate: { directory: 'docs/contributing' } }] },
];
