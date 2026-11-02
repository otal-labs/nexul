import { clsx, type ClassValue } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

// The type scale's own steps (index.css @theme), so a size is never merged away as a colour.
const twMerge = extendTailwindMerge({
  extend: {
    theme: {
      text: ["initial", "micro", "row", "empty-headline", "title", "headline"],
      "font-weight": ["book"],
    },
  },
});

export const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

// Ref callback for the highlighted option of a scrollable list: keeps it visible as the selection moves.
export const keepInView = (el: HTMLElement | null) => el?.scrollIntoView({ block: "nearest" });

// Up to two initials from a display name, e.g. "bob" -> "B", "Jane Doe" -> "JD".
export const initials = (name: string) =>
  name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase() ?? "")
    .join("");
