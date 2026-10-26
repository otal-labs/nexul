import { createElement } from 'react';
import type * as PageTree from 'fumadocs-core/page-tree';
import { contributingGroup, guideGroups } from './sidebar';

export interface DocPage {
	id: string;
	title: string;
	description: string;
	label?: string;
	order?: number;
}

export const docUrl = (id: string) => `/${id}/`;

const byOrder = (a: DocPage, b: DocPage) =>
	(a.order ?? Number.MAX_SAFE_INTEGER) - (b.order ?? Number.MAX_SAFE_INTEGER) || a.id.localeCompare(b.id);

export function docsGroups(pages: DocPage[]) {
	const byId = new Map(pages.map((page) => [page.id, page]));
	const find = (id: string) => {
		const page = byId.get(id);
		if (!page) throw new Error(`src/lib/sidebar.ts lists ${id}, but no page has that id`);
		return page;
	};
	const contributing = pages.filter((page) => page.id.startsWith(contributingGroup.directory)).toSorted(byOrder);

	return [
		...guideGroups.map((group) => ({ label: group.label, icon: group.icon, pages: group.items.map(find) })),
		{ label: contributingGroup.label, icon: contributingGroup.icon, pages: contributing },
	];
}

// The most columns, up to max, that leave no row holding a single card; max again when no count does.
export function gridColumns(count: number, max: number) {
	for (let columns = Math.min(max, count); columns > 1; columns--) if (count % columns !== 1) return columns;
	return Math.min(max, count);
}

export function docsTree(pages: DocPage[]): PageTree.Root {
	return {
		name: 'Docs',
		children: [
			{ type: 'page', name: 'Documentation', url: docUrl('docs') },
			...docsGroups(pages).flatMap((group): PageTree.Node[] => [
				{ type: 'separator', name: group.label, icon: createElement(group.icon) },
				...group.pages.map((page): PageTree.Node => ({ type: 'page', name: page.label ?? page.title, url: docUrl(page.id) })),
			]),
		],
	};
}
