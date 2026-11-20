import { type ClassValue, clsx } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

// The type scale's own steps (global.css @theme), so a size is never merged away as a colour.
const twMerge = extendTailwindMerge({
  extend: {
    theme: {
      text: ["initial", "micro", "row", "body", "sheet-title", "empty-headline", "title", "screen-title", "screen-title-md", "screen-title-sm", "connect-headline"],
    },
  },
});

export const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));
