import type { Element, ElementContent, Root, RootContent } from 'hast';

const svg = (children: ElementContent[], className: string): Element => ({
	type: 'element',
	tagName: 'svg',
	properties: { className: [className], width: 16, height: 16, viewBox: '0 0 16 16', fill: 'none', stroke: 'currentColor', strokeWidth: 1.5, ariaHidden: 'true' },
	children,
});
const shape = (tagName: string, properties: Element['properties']): Element => ({ type: 'element', tagName, properties, children: [] });

// The home page's copy marks: two overlapping squares, and the check that replaces them once copied.
const copyIcon = () => svg([shape('rect', { x: 5.5, y: 5.5, width: 8, height: 8, rx: 1.5 }), shape('path', { d: 'M10.5 5.5v-2a1 1 0 0 0-1-1h-6a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h2' })], 'icon-copy');
const doneIcon = () => svg([shape('path', { d: 'M3 8.5l3 3 7-7' })], 'icon-done');

// Each highlighted block becomes a console with its copy button beside it, built into the page so it is there before
// any script runs; src/lib/code-copy.ts wires the buttons.
const wrap = (pre: Element): Element => ({
	type: 'element',
	tagName: 'div',
	properties: { className: ['code-block'] },
	children: [
		pre,
		{
			type: 'element',
			tagName: 'button',
			properties: { type: 'button', className: ['code-copy'], dataCopyCode: '', ariaLabel: 'Copy code', title: 'Copy code', disabled: true },
			children: [copyIcon(), doneIcon()],
		},
		{ type: 'element', tagName: 'span', properties: { className: ['copy-status'], role: 'status', ariaLive: 'polite', ariaAtomic: 'true' }, children: [] },
	],
});

// Shiki writes its class as a string (`class`), hast elements built elsewhere as a list (`className`).
const classes = (node: Element) => [node.properties.className, node.properties.class].flat().join(' ').split(/\s+/);

const isCodeBlock = (node: RootContent | ElementContent): node is Element => node.type === 'element' && node.tagName === 'pre' && classes(node).includes('shiki');

// rehypeCode puts each highlighted block back as a nested root, so the walk enters roots as well as elements.
function walk(parent: Root | Element) {
	parent.children = parent.children.map((child) => {
		if (isCodeBlock(child)) return wrap(child);
		const node = child as RootContent | Root;
		if (node.type === 'element' || node.type === 'root') walk(node);
		return child;
	}) as typeof parent.children;
}

export const rehypeCodeCopy = () => (tree: Root) => walk(tree);
