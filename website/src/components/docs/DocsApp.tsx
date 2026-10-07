import { useMemo, type ReactNode } from 'react';
import { RootProvider } from 'fumadocs-ui/provider/astro';
import { DocsLayout } from 'fumadocs-ui/layouts/docs';
import { DocsBody, DocsPage, EditOnGitHub } from 'fumadocs-ui/layouts/docs/page';
import type { TOCItemType } from 'fumadocs-core/toc';
import { docsTree, type DocPage } from '../../lib/docs-tree';
import { DocsHeader } from './DocsHeader';
import { DrawerLinks } from './DrawerLinks';
import { PageHeading, type PageHeadingProps } from './PageHeading';
import { SearchDialog } from './SearchDialog';

interface Props {
	pathname: string;
	pages: DocPage[];
	toc: TOCItemType[];
	heading: PageHeadingProps;
	editUrl: string;
	home: boolean;
	children: ReactNode;
}

const noTitle = () => null;

export function DocsApp({ pathname, pages, toc, heading, editUrl, home, children }: Props) {
	const tree = useMemo(() => docsTree(pages), [pages]);

	return (
		<RootProvider pathname={pathname} theme={{ hotKey: false }} search={{ SearchDialog }}>
			<DocsLayout
				tree={tree}
				searchToggle={{ enabled: false }}
				themeSwitch={{ enabled: false }}
				sidebar={{ collapsible: false, footer: <DrawerLinks /> }}
				slots={{ header: DocsHeader, navTitle: noTitle }}
				containerProps={{ style: { paddingTop: 'var(--site-header-height)' } }}
			>
				<DocsPage toc={toc} full={home} breadcrumb={{ enabled: false }}>
					<PageHeading {...heading} />
					<DocsBody>{children}</DocsBody>
					<EditOnGitHub href={editUrl} className="self-start">
						Edit this page
					</EditOnGitHub>
				</DocsPage>
			</DocsLayout>
		</RootProvider>
	);
}
