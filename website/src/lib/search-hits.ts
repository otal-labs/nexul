import type { SortedResult } from 'fumadocs-core/search';

export interface TextPart {
	text: string;
	match: boolean;
}

// A hit's markdown as the words a reader sees: no emphasis marks, link targets or code ticks; an underscore inside a
// word (deploy_list) stays.
export function plainText(markdown: string) {
	return markdown
		.replace(/!?\[([^\]]*)\]\([^)]*\)/g, '$1')
		.replace(/`/g, '')
		.replace(/\*\*|__|(?<!\w)[*_]|[*_](?!\w)/g, '')
		.replace(/\s+/g, ' ')
		.trim();
}

const terms = (query: string) =>
	query
		.trim()
		.split(/\s+/)
		.filter(Boolean)
		.map((term) => term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));

// Splits text into matched and unmatched runs, case-insensitively; a match runs to the end of its word (deploy → deploys).
export function markMatches(text: string, query: string): TextPart[] {
	const words = terms(query);
	if (words.length === 0) return [{ text, match: false }];
	// split with one capturing group puts every match at an odd index.
	return text
		.split(new RegExp(`((?:${words.join('|')})\\w*)`, 'i'))
		.map((part, index) => ({ text: part, match: index % 2 === 1 }))
		.filter((part) => part.text !== '');
}

// The text from a few words before the first match, so a one-line excerpt shows the match instead of the paragraph's start.
export function excerpt(text: string, query: string, lead = 4) {
	const words = terms(query);
	if (words.length === 0) return text;
	const at = text.search(new RegExp(words.join('|'), 'i'));
	if (at <= 0) return text;
	const before = text.slice(0, at).split(' ');
	if (before.length <= lead + 1) return text;
	return `…${before.slice(-lead - 1).join(' ')}${text.slice(at)}`;
}

export interface PageHits {
	page: SortedResult;
	hits: SortedResult[];
}

// Results as pages with at most max hits each, one per section; table rows are left out, since a row out of its table
// reads as noise, a page left with no hit stays only if its title matched, and pages whose title matched come first.
export function groupByPage(items: SortedResult[], query: string, max = 3): PageHits[] {
	const groups: PageHits[] = [];
	for (const item of items) {
		if (item.type === 'page') {
			groups.push({ page: item, hits: [] });
			continue;
		}
		const group = groups.at(-1);
		if (!group || item.table || group.hits.length >= max) continue;
		if (group.hits.some((hit) => hit.url === item.url)) continue;
		group.hits.push(item);
	}
	const words = terms(query);
	if (words.length === 0) return groups;
	const titled = new RegExp(words.join('|'), 'i');
	const named = (group: PageHits) => titled.test(String(group.page.content));
	return groups.filter((group) => group.hits.length > 0 || named(group)).sort((a, b) => Number(named(b)) - Number(named(a)));
}
