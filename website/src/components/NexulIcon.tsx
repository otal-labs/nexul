// Nexul hub glyph, inline so it takes the surrounding text color. Same glyph as public/favicon.svg, without the tile.
export function NexulIcon() {
	return (
		<svg
			className="nexul-icon"
			style={{ width: '1.1em', height: '1.1em', verticalAlign: '-0.2em', flexShrink: 0 }}
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			strokeWidth="2.25"
			strokeLinecap="round"
			strokeLinejoin="round"
			aria-hidden="true"
		>
			<circle cx="12" cy="12" r="8.5" />
			<path d="M7.6 7.6L12 12l4.4 4.4" />
			<circle cx="12" cy="12" r="2.8" fill="currentColor" stroke="none" />
			<circle cx="6" cy="6" r="2.2" />
			<circle cx="18" cy="18" r="2.2" />
		</svg>
	);
}
