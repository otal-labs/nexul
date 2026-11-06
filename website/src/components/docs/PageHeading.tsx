import { MarkdownCopyButton } from 'fumadocs-ui/layouts/docs/page';

export interface PageHeadingProps {
	eyebrow?: string;
	title: string;
	description: string;
	markdownUrl?: string;
}

// The group and Copy page share the row above the title, so the title and its lede keep the full measure.
export function PageHeading({ eyebrow, title, description, markdownUrl }: PageHeadingProps) {
	return (
		<header className="docs-heading">
			{(eyebrow || markdownUrl) && (
				<div className="docs-heading-row">
					<p className="docs-eyebrow">{eyebrow}</p>
					{markdownUrl && (
						<MarkdownCopyButton markdownUrl={markdownUrl} className="docs-copy-page">
							Copy page
						</MarkdownCopyButton>
					)}
				</div>
			)}
			<h1 className="docs-title type-display">{title}</h1>
			<p className="docs-lede">{description}</p>
		</header>
	);
}
