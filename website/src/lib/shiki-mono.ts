import type { ThemeRegistration } from 'shiki';

type Ramp = { background: string; text: string; strong: string; muted: string; faint: string };

// Code stays monochrome like the rest of the Mono Console: emphasis comes from the grey ramp, never a hue.
const mono = (name: string, type: 'dark' | 'light', ramp: Ramp): ThemeRegistration => ({
	name,
	type,
	colors: { 'editor.background': ramp.background, 'editor.foreground': ramp.text },
	tokenColors: [
		{ settings: { foreground: ramp.text } },
		{ scope: ['comment', 'punctuation.definition.comment'], settings: { foreground: ramp.faint, fontStyle: 'italic' } },
		{ scope: ['keyword', 'storage', 'entity.name.function', 'support.function', 'entity.name.tag'], settings: { foreground: ramp.strong } },
		{ scope: ['string', 'constant', 'variable.other.constant', 'markup.inline.raw'], settings: { foreground: ramp.muted } },
		{ scope: ['punctuation', 'meta.brace', 'keyword.operator'], settings: { foreground: ramp.faint } },
	],
});

export const monoDark = mono('mono-console-dark', 'dark', {
	background: '#111111',
	text: '#d4d4d4',
	strong: '#f5f5f5',
	muted: '#9a9a9a',
	faint: '#656565',
});

export const monoLight = mono('mono-console-light', 'light', {
	background: '#f6f6f6',
	text: '#2b2b2b',
	strong: '#0a0a0a',
	muted: '#656565',
	faint: '#999999',
});
