import { MarkdownCopyButton } from 'fumadocs-ui/layouts/docs/page';

export interface PageHeadingProps {
	eyebrow?: string;
	title: string;
	description: string;
	markdownUrl?: string;
}

export function PageHeading({ eyebrow, title, description, markdownUrl }: PageHeadingProps) {
	return (
		<header className="flex flex-col gap-3">
			{eyebrow && <p className="text-xs font-medium text-fd-muted-foreground">{eyebrow}</p>}
			<div className="flex flex-wrap items-start justify-between gap-x-6 gap-y-3">
				<h1 className="docs-title type-display min-w-0">{title}</h1>
				{markdownUrl && (
					<MarkdownCopyButton markdownUrl={markdownUrl} className="docs-copy-page mt-1 shrink-0">
						Copy page
					</MarkdownCopyButton>
				)}
			</div>
			<p className="max-w-[65ch] text-base text-pretty text-fd-muted-foreground md:text-lg">{description}</p>
		</header>
	);
}
