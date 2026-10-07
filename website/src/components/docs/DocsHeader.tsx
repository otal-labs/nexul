import { useEffect } from 'react';
import { Menu } from 'lucide-react';
import { usePathname } from 'fumadocs-core/framework';
import { useSearchContext } from 'fumadocs-ui/contexts/search';
import { useDocsLayout } from 'fumadocs-ui/layouts/docs';
import { FullSearchTrigger, SearchTrigger } from 'fumadocs-ui/layouts/shared/slots/search-trigger';
import { ThemeSwitch } from 'fumadocs-ui/layouts/shared/slots/theme-switch';
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
					<FullSearchTrigger className="h-8 w-48 rounded-md bg-fd-card max-md:hidden lg:w-56 xl:w-64" />
					<SearchTrigger className="size-9 justify-center md:hidden" />
					<ThemeSwitch className="rounded-md p-0.5 *:rounded-sm" />
					<SidebarTrigger className="inline-flex size-9 items-center justify-center rounded-md text-fd-muted-foreground hover:bg-fd-accent hover:text-fd-foreground md:hidden [&_svg]:size-5">
						<Menu />
					</SidebarTrigger>
				</>
			}
		/>
	);
}
