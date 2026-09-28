import { expect, test } from 'bun:test';
import { guide, guideGroups } from '../src/lib/sidebar';

test('every guide page sits in exactly one sidebar group', async () => {
	const pages: string[] = [];
	for await (const path of new Bun.Glob('*.{md,mdx}').scan({ cwd: `${import.meta.dir}/../src/content/docs/docs/guide` })) {
		pages.push(guide(path.replace(/\.mdx?$/, '')));
	}
	const listed = guideGroups.flatMap((group) => group.items);

	expect(listed.toSorted()).toEqual(pages.toSorted());
});
