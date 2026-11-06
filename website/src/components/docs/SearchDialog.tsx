import { type CSSProperties, useSyncExternalStore } from 'react';
import {
	SearchDialog as Dialog,
	SearchDialogClose,
	SearchDialogContent,
	SearchDialogHeader,
	SearchDialogIcon,
	SearchDialogInput,
	SearchDialogList,
	SearchDialogListItem,
	SearchDialogOverlay,
	type SharedProps,
} from 'fumadocs-ui/components/dialog/search';
import { useDocsSearch } from 'fumadocs-core/search/client';
import { staticClient } from 'fumadocs-core/search/client/orama-static';
import { excerpt, groupByPage, markMatches, plainText } from '../../lib/search-hits';

// The index is built at build time (src/pages/api/search.ts) and searched in the browser, so the site needs no server.
const client = staticClient();

interface Names {
	headings: Record<string, string>;
	descriptions: Record<string, string>;
}

// A hit knows its section only by anchor, and a page row only its title; src/pages/api/headings.ts fills both in, once.
let names: Names = { headings: {}, descriptions: {} };
let requested = false;
const listeners = new Set<() => void>();
const subscribe = (listener: () => void) => {
	listeners.add(listener);
	if (!requested) {
		requested = true;
		fetch('/api/headings')
			.then((response) => response.json())
			.then((loaded: Names) => {
				names = loaded;
				for (const notify of listeners) notify();
			})
			.catch(() => undefined);
	}
	return () => listeners.delete(listener);
};
const useNames = () => useSyncExternalStore(subscribe, () => names, () => names);

// Where a search that found nothing can start instead.
const startPages = [
	{ href: '/docs/', title: 'Documentation', line: 'Every guide and contributing page' },
	{ href: '/docs/guide/install/', title: 'Install', line: 'Put Nexul on a server with one command' },
	{ href: '/docs/guide/upgrade/', title: 'Upgrade', line: 'Move the instance to a new release' },
];

// A page name's width in its own type (12px Inter), so the section name beside it can take the rest of the line.
let pen: CanvasRenderingContext2D | null = null;
const nameWidth = (text: string) => {
	pen ??= document.createElement('canvas').getContext('2d');
	if (!pen) return 0;
	pen.font = `12px ${getComputedStyle(document.documentElement).getPropertyValue('--font-sans-stack')}`;
	return Math.ceil(pen.measureText(text).width);
};

function Marked({ text, query }: { text: string; query: string }) {
	return markMatches(text, query).map((part, index) => (part.match ? <strong key={index}>{part.text}</strong> : <span key={index}>{part.text}</span>));
}

export function SearchDialog(props: SharedProps) {
	const { search, setSearch, query } = useDocsSearch({ client });
	const { headings, descriptions } = useNames();
	const groups = query.data === 'empty' ? [] : groupByPage(query.data, search);
	const pageOf = new Map(groups.flatMap((group) => group.hits.map((hit) => [hit.id, plainText(String(group.page.content))] as const)));
	// A page whose sections matched shows only those sections; a page that matched by its title alone stays a row.
	const items = query.data === 'empty' ? null : groups.flatMap((group) => (group.hits.length > 0 ? group.hits : [group.page]));

	return (
		<Dialog search={search} onSearchChange={setSearch} isLoading={query.isLoading} {...props}>
			<SearchDialogOverlay className="site-search-overlay" />
			<SearchDialogContent className="site-search-dialog">
				<SearchDialogHeader>
					<SearchDialogIcon />
					<SearchDialogInput />
					<SearchDialogClose className="search-close">
						<span className="search-close-key">ESC</span>
						<svg className="search-close-mark" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
							<path d="M18 6 6 18M6 6l12 12" />
						</svg>
					</SearchDialogClose>
				</SearchDialogHeader>
				<SearchDialogList
					items={items}
					Empty={() => (
						<div className="search-empty">
							<p role="status">
								No results for <code>{search}</code>
							</p>
							<ul className="search-start" aria-label="Start from">
								{startPages.map((start) => (
									<li key={start.href}>
										<a href={start.href}>
											<span className="search-heading">{start.title}</span>
											<span className="search-excerpt">{start.line}</span>
										</a>
									</li>
								))}
							</ul>
						</div>
					)}
					Item={({ item, onClick }) => {
						const text = plainText(String(item.content));
						const page = pageOf.get(item.id);
						// A hit in a page's intro has no anchor, so its page is its heading and is named once.
						const section = item.url.includes('#');
						const heading = item.type === 'text' ? (section && headings[item.url]) || page || text : text;
						const line = (item.type === 'text' && excerpt(text, search)) || (item.type === 'page' && descriptions[item.url]);
						return (
							<SearchDialogListItem item={item} onClick={onClick}>
								<span className="search-line" style={{ '--where': `${page && section ? nameWidth(page) : 0}px` } as CSSProperties}>
									<span className="search-heading">
										<Marked text={heading} query={search} />
									</span>
									{page && section && <span className="search-where">{page}</span>}
								</span>
								{line && (
									<span className="search-excerpt">
										<Marked text={line} query={search} />
									</span>
								)}
							</SearchDialogListItem>
						);
					}}
				/>
			</SearchDialogContent>
		</Dialog>
	);
}
