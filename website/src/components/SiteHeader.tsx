import type { ReactNode } from 'react';
import { NexulIcon } from './NexulIcon';

export const github = 'https://github.com/otal-labs/nexul';

export const siteLinks = [
	{ href: '/roadmap/', text: 'Roadmap' },
	{ href: '/changelog/', text: 'Changelog' },
	{ href: github, text: 'GitHub' },
];

interface Props {
	pathname: string;
	// Docs pages pass their search, theme and menu controls; below 768px they replace the links, which move into the menu.
	docsControls?: ReactNode;
}

export function SiteHeader({ pathname, docsControls }: Props) {
	const current = (href: string) => (pathname.startsWith(href) ? 'page' : undefined);

	return (
		<header className={docsControls ? 'site-header site-header-docs' : 'site-header'}>
			<div className="site-header-inner">
				<a className="site-wordmark" href="/" aria-label="Nexul home">
					<NexulIcon /> Nexul
				</a>
				<nav className="site-nav" aria-label="Main navigation">
					<a href="/docs/" aria-current={current('/docs/')}>
						Docs
					</a>
					{siteLinks.map((link) => (
						<a key={link.href} className="site-nav-detail" href={link.href} aria-current={current(link.href)}>
							{link.text}
						</a>
					))}
				</nav>
				<div className="site-actions">
					{docsControls}
					<a className="site-button" href="/#install">
						Get started
					</a>
				</div>
			</div>
		</header>
	);
}
