
export const guide = (slug: string) => (slug === 'index' ? 'docs/guide' : `docs/guide/${slug}`);

export const guideGroups = [
	{ label: 'Get started', items: [guide('index'), guide('install'), guide('setup-wizard'), guide('github-app'), guide('upgrade')] },
	{ label: 'Deploy', items: [guide('runners'), guide('stacks-and-deploys'), guide('topology-and-dns'), guide('logs')] },
	{
		label: 'Work together',
		items: [
			guide('projects-and-repositories'),
			guide('people-and-access'),
			guide('welcome-wizard'),
			guide('docs-tickets-and-board'),
			guide('chat-and-voice'),
			guide('bots'),
			guide('calendar'),
			guide('automations'),
		],
	},
	{
		label: 'Agents',
		items: [guide('computer-setup'), guide('paired-computers'), guide('plays'), guide('interview'), guide('memories'), guide('mcp-server'), guide('triage')],
	},
	{ label: 'Reference', items: [guide('api-and-tokens'), guide('desktop-app'), guide('phone-app')] },
];

// Contributing pages are not listed by hand: every page in the directory, ordered by its sidebar.order.
export const contributingGroup = { label: 'Contributing', directory: 'docs/contributing' };
