import type { ReactNode } from 'react';
import { Menu, Moon, Sun, X } from 'lucide-react';
import { NexulIcon } from './NexulIcon';

export const github = 'https://github.com/otal-labs/nexul';

export const siteLinks = [
	{ href: '/docs/', text: 'Docs' },
	{ href: '/roadmap/', text: 'Roadmap' },
	{ href: '/changelog/', text: 'Changelog' },
	{ href: github, text: 'GitHub' },
];

const isCurrent = (pathname: string, href: string) => (pathname.startsWith(href) ? 'page' : undefined);

interface Props {
	pathname: string;
	// Docs pages pass their search triggers and the trigger of their own phone menu, the docs sidebar drawer.
	docsControls?: ReactNode;
	menuTrigger?: ReactNode;
}

export function SiteHeader({ pathname, docsControls, menuTrigger }: Props) {
	return (
		<header className="site-header">
			<div className="site-header-inner">
				<a className="site-wordmark" href="/" aria-label="Nexul home">
					<NexulIcon /> Nexul
				</a>
				<nav className="site-nav" aria-label="Main navigation">
					{siteLinks.map((link) => (
						<a key={link.href} href={link.href} aria-current={isCurrent(pathname, link.href)}>
							{link.text}
						</a>
					))}
				</nav>
				<div className="site-actions">
					{docsControls}
					<ThemeToggle />
					<a className="site-button" href="/#install">
						Get started
					</a>
					{menuTrigger ?? (
						<button type="button" className="site-icon-button site-menu-trigger" data-site-menu-open aria-controls="site-menu" aria-expanded="false" aria-label="Open menu">
							<Menu aria-hidden />
						</button>
					)}
				</div>
			</div>
			{!menuTrigger && <SiteMenuDialog pathname={pathname} />}
		</header>
	);
}

// The head script sets .light or .dark on <html> before paint, so the icons switch by CSS and the markup never depends on the theme.
function ThemeToggle() {
	return (
		<button type="button" className="site-icon-button theme-toggle" data-theme-toggle aria-label="Dark theme">
			<Moon className="theme-icon theme-icon-dark" aria-hidden />
			<Sun className="theme-icon theme-icon-light" aria-hidden />
		</button>
	);
}

export function SiteMenuLinks({ pathname }: { pathname: string }) {
	return (
		<nav aria-label="Site" className="site-menu-links">
			{siteLinks.map((link) => (
				<a key={link.href} href={link.href} aria-current={isCurrent(pathname, link.href)}>
					{link.text}
				</a>
			))}
			<a className="site-button site-menu-cta" href="/#install">
				Get started
			</a>
		</nav>
	);
}

function SiteMenuDialog({ pathname }: { pathname: string }) {
	return (
		<dialog id="site-menu" className="site-sheet" aria-label="Menu">
			<div className="site-sheet-top">
				<button type="button" className="site-icon-button" data-site-menu-close aria-label="Close menu">
					<X aria-hidden />
				</button>
			</div>
			<SiteMenuLinks pathname={pathname} />
		</dialog>
	);
}
