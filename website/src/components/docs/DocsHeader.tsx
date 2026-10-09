import { useEffect } from 'react';
import { Menu } from 'lucide-react';
import { usePathname } from 'fumadocs-core/framework';
import { useSearchContext } from 'fumadocs-ui/contexts/search';
import { useDocsLayout } from 'fumadocs-ui/layouts/docs';
import { FullSearchTrigger, SearchTrigger } from 'fumadocs-ui/layouts/shared/slots/search-trigger';
import { SiteHeader } from '../SiteHeader';

const isTyping = (target: EventTarget | null) =>
	target instanceof HTMLElement && (target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName));

// Fumadocs binds one search shortcut (Cmd/Ctrl+K); "/" is the other one readers expect.
function useSlashOpensSearch() {
	const { setOpenSearch } = useSearchContext();
	useEffect(() => {
		const onKeyDown = (event: KeyboardEvent) => {
			if (event.key !== '/' || event.defaultPrevented || isTyping(event.target)) return;
			event.preventDefault();
			setOpenSearch(true);
		};
		window.addEventListener('keydown', onKeyDown);
		return () => window.removeEventListener('keydown', onKeyDown);
	}, [setOpenSearch]);
}

export function DocsHeader() {
	const { slots } = useDocsLayout();
	const SidebarTrigger = slots.sidebar.trigger;
	useSlashOpensSearch();

	return (
		<SiteHeader
			pathname={usePathname()}
			docsControls={
				<>
					<FullSearchTrigger className="site-search-full" />
					<SearchTrigger className="site-icon-button site-search-icon" />
				</>
			}
			menuTrigger={
				<SidebarTrigger className="site-icon-button site-menu-trigger">
					<Menu aria-hidden />
				</SidebarTrigger>
			}
		/>
	);
}
