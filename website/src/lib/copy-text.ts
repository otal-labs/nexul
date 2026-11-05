// The copy feedback every copy button on the site shares: the icon swap through data-copied, "Copied" announced in its
// status, a failure explained in place, and the button back at rest after 1.4s.

const resetTimers = new WeakMap<HTMLButtonElement, number>();
const copyAttempts = new WeakMap<HTMLButtonElement, number>();

export async function copyWithFeedback(button: HTMLButtonElement, text: string, status: HTMLElement, label: string, failure: string) {
	const attempt = (copyAttempts.get(button) ?? 0) + 1;
	copyAttempts.set(button, attempt);

	const resetTimer = resetTimers.get(button);
	if (resetTimer) window.clearTimeout(resetTimer);

	try {
		if (!navigator.clipboard?.writeText) throw new Error('Clipboard API unavailable');
		await navigator.clipboard.writeText(text);
		if (copyAttempts.get(button) !== attempt) return;
		button.dataset.copied = '';
		delete button.dataset.copyFailed;
		button.title = 'Copied';
		status.textContent = 'Copied';
	} catch {
		if (copyAttempts.get(button) !== attempt) return;
		delete button.dataset.copied;
		button.dataset.copyFailed = '';
		button.title = 'Copy failed';
		status.textContent = failure;
	}

	resetTimers.set(
		button,
		window.setTimeout(() => {
			delete button.dataset.copied;
			delete button.dataset.copyFailed;
			button.title = label;
			status.textContent = '';
		}, 1400),
	);
}

// A scroller wider than its box marks data-overflow, and data-at-end once scrolled to its end, so CSS fades the cut edge
// only while there is more to see.
export function markOverflow(scrollers: Iterable<HTMLElement>) {
	const measure = (element: HTMLElement) => {
		element.toggleAttribute('data-overflow', element.scrollWidth > element.clientWidth + 1);
		element.toggleAttribute('data-at-end', element.scrollLeft + element.clientWidth >= element.scrollWidth - 1);
	};
	const resized = new ResizeObserver((entries) => entries.forEach((entry) => measure(entry.target as HTMLElement)));
	for (const element of scrollers) {
		resized.observe(element);
		element.addEventListener('scroll', () => measure(element), { passive: true });
	}
}
