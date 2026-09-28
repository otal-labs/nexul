import { expect, test } from 'bun:test';
import { fetchReleases, parseNotes, toRelease } from '../src/lib/releases';

const body = `## What's Changed
* Cut a beta once a day when master has moved by @Onik97 in https://github.com/otal-labs/nexul/pull/65
* Explain what "by @someone in" means by @Onik97 in https://github.com/otal-labs/nexul/pull/66

## New Contributors
* @someone made their first contribution in https://github.com/otal-labs/nexul/pull/64

**Full Changelog**: https://github.com/otal-labs/nexul/compare/v0.2.0-beta.2...v0.2.0-beta.3
`;

test('parses pull request lines and the compare link', () => {
	expect(parseNotes(body)).toEqual({
		changes: [
			{ title: 'Cut a beta once a day when master has moved', number: 65, url: 'https://github.com/otal-labs/nexul/pull/65' },
			{ title: 'Explain what "by @someone in" means', number: 66, url: 'https://github.com/otal-labs/nexul/pull/66' },
		],
		compare: { url: 'https://github.com/otal-labs/nexul/compare/v0.2.0-beta.2...v0.2.0-beta.3', previousTag: 'v0.2.0-beta.2' },
	});
});

test('an empty or hand-written body yields no changes and no compare link', () => {
	expect(parseNotes('')).toEqual({ changes: [], compare: undefined });
	expect(parseNotes('First release.')).toEqual({ changes: [], compare: undefined });
});

test('a CRLF body parses the same as LF', () => {
	expect(parseNotes(body.replaceAll('\n', '\r\n')).changes).toHaveLength(2);
});

test('maps the API shape and keeps only a commit sha as the sha', () => {
	const api = {
		tag_name: 'v0.2.0',
		html_url: 'https://github.com/otal-labs/nexul/releases/tag/v0.2.0',
		published_at: '2026-09-26T08:37:46Z',
		draft: false,
		target_commitish: '5aa17c7547fdf6ef65ff71d0c479ce72999e4078',
		body: null,
	};
	expect(toRelease(api)).toMatchObject({ tag: 'v0.2.0', sha: '5aa17c7547fdf6ef65ff71d0c479ce72999e4078', changes: [] });
	expect(toRelease({ ...api, target_commitish: 'master' }).sha).toBeUndefined();
});

test('lists releases newest first by publish time, not in GitHub\'s tag-text order', async () => {
	const release = (tag: string, published_at: string) => ({ tag_name: tag, html_url: '', published_at, draft: false, target_commitish: 'master', body: '' });
	const original = globalThis.fetch;
	globalThis.fetch = (async () =>
		Response.json([
			release('v0.2.0-beta.9', '2026-09-28T10:51:32Z'),
			release('v0.2.0-beta.11', '2026-09-28T22:27:55Z'),
			release('v0.2.0-beta.10', '2026-09-28T15:17:33Z'),
		])) as unknown as typeof fetch;
	try {
		expect((await fetchReleases('otal-labs/nexul')).map((r) => r.tag)).toEqual(['v0.2.0-beta.11', 'v0.2.0-beta.10', 'v0.2.0-beta.9']);
	} finally {
		globalThis.fetch = original;
	}
});
