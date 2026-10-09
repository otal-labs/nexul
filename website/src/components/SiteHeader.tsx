import type { ReactNode } from 'react';
import { Menu, Moon, Sun, X } from 'lucide-react';
import { NexulIcon } from './NexulIcon';

export const github = 'https://github.com/otal-labs/nexul';

export const siteLinks = [
	{ href: '/docs/', text: 'Docs' },
	{ href: '/roadmap/', text: 'Roadmap' },
	{ href: '/changelog/', text: 'Changelog' },
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
					<a className="site-icon-button site-github" href={github} aria-label="GitHub" title="GitHub">
						<GithubMark />
					</a>
					<ThemeToggle />
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
		<button type="button" className="site-icon-button theme-toggle" data-theme-toggle aria-label="Dark theme" title="Dark theme">
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
			<a className="site-menu-external" href={github}>
				<GithubMark />
				GitHub
			</a>
		</nav>
	);
}

// lucide dropped brand marks, so GitHub's is inlined.
function GithubMark() {
	return (
		<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
			<path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
		</svg>
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
