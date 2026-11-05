import { copyWithFeedback, markOverflow } from './copy-text';

// The docs' code blocks, as src/lib/rehype-code-copy.ts builds them: a copy button each, and a faded edge while a long
// line runs past the box.
const blocks = document.querySelectorAll<HTMLElement>('.code-block');

markOverflow([...blocks].flatMap((block) => block.querySelector<HTMLElement>('pre') ?? []));

for (const block of blocks) {
	const button = block.querySelector<HTMLButtonElement>('[data-copy-code]');
	const status = block.querySelector<HTMLElement>('.copy-status');
	const code = block.querySelector('pre code');
	if (!button || !status || !code) continue;
	button.disabled = false;
	button.addEventListener('click', () => copyWithFeedback(button, code.textContent ?? '', status, 'Copy code', 'Copy failed. Select and copy the code manually.'));
}
