import { Bot, Code, GitPullRequest, Rocket, Server, Users } from 'lucide-react';

export const guide = (slug: string) => (slug === 'index' ? 'docs/guide' : `docs/guide/${slug}`);

export const guideGroups = [
	{ label: 'Get started', icon: Rocket, items: [guide('index'), guide('install'), guide('setup-wizard'), guide('github-app'), guide('upgrade')] },
	{ label: 'Deploy', icon: Server, items: [guide('runners'), guide('stacks-and-deploys'), guide('topology-and-dns'), guide('logs')] },
	{
		label: 'Work together',
		icon: Users,
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
		icon: Bot,
		items: [guide('computer-setup'), guide('paired-computers'), guide('plays'), guide('interview'), guide('memories'), guide('mcp-server')],
	},
	{ label: 'Reference', icon: Code, items: [guide('api-and-tokens'), guide('desktop-app'), guide('phone-app')] },
];

// Contributing pages are not listed by hand: every page in the directory, ordered by its sidebar.order.
export const contributingGroup = { label: 'Contributing', icon: GitPullRequest, directory: 'docs/contributing' };
