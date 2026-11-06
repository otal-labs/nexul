import { expect, test } from 'bun:test';
import { guideGroups } from '../src/lib/sidebar';
import { docUrl, docsTree } from '../src/lib/docs-tree';

test('the docs home is a row of the sidebar, so it is marked when open', () => {
	const pages = guideGroups.flatMap((group) => group.items).map((id) => ({ id, title: id, description: '' }));
	const urls = docsTree(pages).children.flatMap((node) => ('url' in node ? [node.url] : []));

	expect(urls).toContain(docUrl('docs'));
});
