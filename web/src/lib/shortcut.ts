// Apple keyboards say ⌘ where everyone else says Ctrl.
export const isApplePlatform = () => /Mac|iPhone|iPad/.test(navigator.userAgent);

export const commandPaletteShortcut = () => (isApplePlatform() ? "⌘K" : "Ctrl K");
