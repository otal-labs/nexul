import { getCollection, type CollectionEntry } from 'astro:content';
import type { DocPage } from './docs-tree';

export const docEntries = () => getCollection('docs');

export const toDocPage = ({ id, data }: CollectionEntry<'docs'>): DocPage => ({
	id,
	title: data.title,
	description: data.description,
	label: data.sidebar?.label,
	order: data.sidebar?.order,
});

// What "Copy page" and /<page>.md hand to a reader or an agent: the page as written, under its title and lead.
export const pageMarkdown = ({ data, body }: CollectionEntry<'docs'>) => `# ${data.title}\n\n${data.description}\n\n${body ?? ''}`;
