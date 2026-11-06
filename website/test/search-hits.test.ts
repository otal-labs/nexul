import { expect, test } from 'bun:test';
import type { SortedResult } from 'fumadocs-core/search';
import { excerpt, groupByPage, markMatches, plainText } from '../src/lib/search-hits';

const hit = (type: SortedResult['type'], url: string, extra: Partial<SortedResult> = {}): SortedResult => ({ id: `${type}:${url}:${extra.table ?? ''}`, type, url, content: url, ...extra });

test('a hit reads as plain words, and a tool name keeps its underscore', () => {
	expect(plainText('Run **`deploy_list`** with [the API](/docs/api/) or *stack_get*.')).toBe('Run deploy_list with the API or stack_get.');
});

test('an excerpt starts a few words before the first match, so a one-line row shows it', () => {
	const text = 'Every part of Nexul runs as a native service under the system service manager and the runner deploys stacks';

	expect(excerpt(text, 'deploys')).toBe('…manager and the runner deploys stacks');
	expect(excerpt('The runner deploys stacks', 'deploys')).toBe('The runner deploys stacks');
});

test('matches are marked case-insensitively, every term of the query', () => {
	expect(markMatches('Deploy a stack, then the stacks deployed', 'deploy stack').filter((part) => part.match).map((part) => part.text)).toEqual([
		'Deploy',
		'stack',
		'stacks',
		'deployed',
	]);
});

test('results keep each page with at most three of its sections, one hit per section and no table rows', () => {
	const page = '/docs/guide/runners/';
	const groups = groupByPage(
		[
			hit('page', page),
			hit('text', `${page}#add`),
			hit('text', `${page}#add`),
			hit('text', `${page}#roles`, { table: 'table-0' }),
			hit('heading', `${page}#remove`),
			hit('text', `${page}#scale`),
			hit('text', `${page}#logs`),
		],
		'runner',
	);

	expect(groups.map((group) => [group.page.url, group.hits.map((item) => item.url)])).toEqual([[page, [`${page}#add`, `${page}#remove`, `${page}#scale`]]]);
});

test('a page left with no hit stays only when its title matches', () => {
	const groups = groupByPage(
		[
			hit('page', '/docs/guide/logs/', { content: 'Logs' }),
			hit('page', '/docs/contributing/layout/', { content: 'Repository layout' }),
			hit('text', '/docs/contributing/layout/#tree', { table: 'table-0' }),
		],
		'logs',
	);

	expect(groups.map((group) => group.page.url)).toEqual(['/docs/guide/logs/']);
});

test('pages whose title matches come before pages that matched in their text, each keeping its order', () => {
	const groups = groupByPage(
		[
			hit('page', '/docs/guide/install/', { content: 'Install' }),
			hit('text', '/docs/guide/install/#linux'),
			hit('page', '/docs/guide/runners/', { content: 'Runners' }),
			hit('text', '/docs/guide/runners/#add'),
			hit('page', '/docs/guide/logs/', { content: 'Logs' }),
			hit('text', '/docs/guide/logs/#runner'),
			hit('page', '/docs/guide/personal-runners/', { content: 'Personal runners' }),
		],
		'runner',
	);

	expect(groups.map((group) => group.page.url)).toEqual(['/docs/guide/runners/', '/docs/guide/personal-runners/', '/docs/guide/install/', '/docs/guide/logs/']);
});
