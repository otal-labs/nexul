export interface Change {
	title: string;
	number: number;
	url: string;
	dependency: boolean;
}

export interface Release {
	tag: string;
	url: string;
	publishedAt: Date;
	sha?: string;
	changes: Change[];
	compare?: { url: string; previousTag: string };
}

interface ApiRelease {
	tag_name: string;
	html_url: string;
	published_at: string | null;
	draft: boolean;
	target_commitish: string;
	body: string | null;
}

const changeLine = /^\* (.+) by @(\S+) in (https:\/\/github\.com\/\S+\/pull\/(\d+))$/;
const compareLine = /^\*\*Full Changelog\*\*: (https:\/\/github\.com\/\S+\/compare\/(\S+)\.\.\.\S+)$/;

export function parseNotes(body: string): Pick<Release, 'changes' | 'compare'> {
	const changes: Change[] = [];
	let compare: Release['compare'];
	for (const line of body.split(/\r?\n/)) {
		const change = changeLine.exec(line.trim());
		if (change) {
			changes.push({ title: change[1]!, url: change[3]!, number: Number(change[4]), dependency: change[2] === 'dependabot[bot]' });
			continue;
		}
		const link = compareLine.exec(line.trim());
		if (link) compare = { url: link[1]!, previousTag: link[2]! };
	}
	return { changes, compare };
}

export function toRelease(api: ApiRelease): Release {
	return {
		tag: api.tag_name,
		url: api.html_url,
		publishedAt: new Date(api.published_at ?? 0),
		sha: /^[0-9a-f]{40}$/.test(api.target_commitish) ? api.target_commitish : undefined,
		...parseNotes(api.body ?? ''),
	};
}

const nextPage = (link: string | null) => link?.match(/<([^>]+)>;\s*rel="next"/)?.[1];

export async function fetchReleases(repo: string): Promise<Release[]> {
	const headers: Record<string, string> = { Accept: 'application/vnd.github+json' };
	if (process.env.GITHUB_TOKEN) headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN}`;
	const releases: ApiRelease[] = [];
	let url: string | undefined = `https://api.github.com/repos/${repo}/releases?per_page=100`;
	while (url) {
		const response = await fetch(url, { headers });
		if (!response.ok) throw new Error(`GET ${repo} releases failed: ${response.status}`);
		releases.push(...((await response.json()) as ApiRelease[]));
		url = nextPage(response.headers.get('link'));
	}
	// phone-v* and android-v* releases are the phone app's; GitHub lists by day then tag text, so sort by publish time.
	return releases
		.filter((release) => !release.draft && release.tag_name.startsWith('v'))
		.map(toRelease)
		.sort((a, b) => b.publishedAt.getTime() - a.publishedAt.getTime());
}
