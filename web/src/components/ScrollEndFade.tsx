import { cn } from "@/lib/utils";

interface ScrollEndFadeProps {
  // The surface it fades into, such as from-card or from-popover, and any bleed to match the rows.
  className?: string;
}

// Last child of a scroller: its bottom 24px fade while rows lie past it, and nothing once scrolled to the end or when it fits.
export const ScrollEndFade = ({ className }: ScrollEndFadeProps) => (
  <div aria-hidden className={cn("scroll-end-fade pointer-events-none sticky bottom-0 left-0 z-20 -mt-6 h-6 bg-linear-to-t from-card to-transparent", className)} />
);
