// Centred dialog from 640px up, a full-screen sheet below it.
export const PAIR_DIALOG_FRAME =
  "flex max-h-[min(90dvh,52rem)] flex-col gap-0 p-0 sm:max-w-[min(48rem,calc(100%-2rem))] max-sm:inset-0 max-sm:top-0 max-sm:left-0 max-sm:h-dvh max-sm:max-h-none max-sm:max-w-none max-sm:translate-x-0 max-sm:translate-y-0 max-sm:rounded-none max-sm:border-0";

// 17px over the title puts its cap height level with the corner close button.
export const PAIR_DIALOG_HEADER = "gap-3 border-b border-border px-4 pt-[17px] pb-4 text-left sm:px-6";

// Set up holds a fixed height from md up, so opening a provider's transcript scrolls the body instead of resizing the dialog.
export const SETUP_DIALOG_FRAME = "md:h-[min(90dvh,46rem)] md:max-w-[min(55rem,calc(100%-3rem))]";

// Set up's body holds the run still and scrolls only the provider list.
export const SETUP_BODY = "flex min-h-0 flex-1 flex-col px-4 pt-4 sm:px-6";

export const SETUP_LEAD = "Each provider you switch on runs one short turn to connect Nexul and install its skills.";
