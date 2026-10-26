import { expect, test } from 'bun:test';
import { guideGroups } from '../src/lib/sidebar';
import { docUrl, docsTree, gridColumns } from '../src/lib/docs-tree';

test('a card grid never leaves a row holding one card when a column count avoids it', () => {
	for (const max of [3, 4]) {
		for (let count = 2; count <= 12; count++) {
			const columns = gridColumns(count, max);
			const avoidable = Array.from({ length: Math.min(max, count) - 1 }, (_, index) => index + 2).some((c) => count % c !== 1);
			if (avoidable) expect(count % columns).not.toBe(1);
			expect(columns).toBeLessThanOrEqual(max);
		}
	}
});

test('the docs home is a row of the sidebar, so it is marked when open', () => {
	const pages = guideGroups.flatMap((group) => group.items).map((id) => ({ id, title: id, description: '' }));
	const urls = docsTree(pages).children.flatMap((node) => ('url' in node ? [node.url] : []));

	expect(urls).toContain(docUrl('docs'));
});
