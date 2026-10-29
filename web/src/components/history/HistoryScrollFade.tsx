import { cn } from "@/lib/utils";

interface HistoryScrollFadeProps {
  /** Which edge of the scroller it sits on; the top one hangs under a sticky header 44px tall. */
  edge: "top" | "bottom";
  /** 16px for a list of rows, 28px where a heading may land under it. */
  size?: "sm" | "lg";
  /** The top fade sits at the scroller's own top instead of under a sticky header. */
  flush?: boolean;
}

// Where a scrolling column meets another surface, its text fades out instead of being cut on a flat edge. The fade shows only
// while there is more to scroll to (the scroll-fade classes in index.css), so the column at rest is not washed out.
export const HistoryScrollFade = ({ edge, size = "sm", flush = false }: HistoryScrollFadeProps) => (
  <div
    aria-hidden
    className={cn(
      "pointer-events-none sticky shrink-0",
      size === "lg" ? "[--fade:28px]" : "[--fade:16px]",
      edge === "top" && "scroll-fade-top z-[5] bg-linear-to-b from-popover to-transparent",
      edge === "top" && (flush ? "top-0" : "top-11"),
      edge === "top" && (size === "lg" ? "-mb-7 h-7" : "-mb-4 h-4"),
      edge === "bottom" && "scroll-fade-bottom bottom-0 -mt-4 h-4 bg-linear-to-t from-popover to-transparent",
    )}
  />
);
