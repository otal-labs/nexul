import type { ThemeRegistration } from 'shiki';

// Code is the app's console in both modes: dark, with emphasis from a grey ramp and never a hue, so no token reads as
// state or as the ember's "act here".
export const codeTheme: ThemeRegistration = {
	name: 'nexul-console',
	type: 'dark',
	colors: { 'editor.background': '#0b0b0b', 'editor.foreground': '#d4d4d4' },
	tokenColors: [
		{ settings: { foreground: '#d4d4d4' } },
		{ scope: ['comment', 'punctuation.definition.comment'], settings: { foreground: '#737373', fontStyle: 'italic' } },
		{ scope: ['keyword', 'storage', 'entity.name.tag', 'entity.name.function', 'support.function', 'entity.name.type'], settings: { foreground: '#f5f5f5' } },
		{ scope: ['string', 'constant', 'variable.other.constant', 'markup.inline.raw'], settings: { foreground: '#a3a3a3' } },
		{ scope: ['punctuation', 'meta.brace', 'keyword.operator'], settings: { foreground: '#737373' } },
	],
};
