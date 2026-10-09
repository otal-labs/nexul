// The header's controls on every page, without a framework so the landing pages ship no React: the theme toggle, the
// phone menu dialog, and the input mode that lets keyboard-opened overlays skip their motion.

const root = document.documentElement;
const media = matchMedia('(prefers-color-scheme: dark)');

const syncToggles = () => {
	const dark = root.classList.contains('dark');
	for (const button of document.querySelectorAll<HTMLButtonElement>('[data-theme-toggle]')) {
		button.setAttribute('aria-pressed', String(dark));
	}
};

const applyTheme = (dark: boolean) => {
	root.classList.toggle('dark', dark);
	root.classList.toggle('light', !dark);
	syncToggles();
};

media.addEventListener('change', (event) => {
	if (localStorage.getItem('theme') === 'light' || localStorage.getItem('theme') === 'dark') return;
	applyTheme(event.matches);
});

const openMenu = () => {
	const dialog = document.querySelector<HTMLDialogElement>('#site-menu');
	if (!dialog) return;
	dialog.showModal();
	document.querySelector('[data-site-menu-open]')?.setAttribute('aria-expanded', 'true');
};

const closeMenu = (dialog: HTMLDialogElement) => {
	dialog.close();
};

document.addEventListener('click', (event) => {
	const target = event.target as Element;
	if (target.closest('[data-theme-toggle]')) {
		const dark = !root.classList.contains('dark');
		localStorage.setItem('theme', dark ? 'dark' : 'light');
		applyTheme(dark);
		return;
	}
	if (target.closest('[data-site-menu-open]')) {
		openMenu();
		return;
	}
	const dialog = target.closest<HTMLDialogElement>('#site-menu');
	if (!dialog) return;
	// A click on the dialog element itself landed on its backdrop.
	if (target === dialog || target.closest('[data-site-menu-close]')) closeMenu(dialog);
});

document.addEventListener('close', (event) => {
	if ((event.target as Element).id !== 'site-menu') return;
	document.querySelector('[data-site-menu-open]')?.setAttribute('aria-expanded', 'false');
}, true);

// The phone menu has nothing to show from 768px, where the header lists every link.
matchMedia('(min-width: 768px)').addEventListener('change', (event) => {
	const dialog = document.querySelector<HTMLDialogElement>('#site-menu');
	if (event.matches && dialog?.open) closeMenu(dialog);
});

window.addEventListener('keydown', () => (root.dataset.input = 'key'), true);
window.addEventListener('pointerdown', () => (root.dataset.input = 'pointer'), true);

syncToggles();
