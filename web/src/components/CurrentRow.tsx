import { cn } from "@/lib/utils";

// One "you are here" in every palette: hover keeps the accent lift, so hover and current never read alike.
export const currentRowFillClass = "bg-brand/12 dark:bg-brand/16";

export const currentRowClass = cn(currentRowFillClass, "font-medium text-foreground");

// The row must be `relative`; inside a sliding `ActiveIndicator` it moves with the block.
export const CurrentRowEdge = ({ className }: { className?: string }) => (
  <span aria-hidden className={cn("pointer-events-none absolute inset-y-2 left-0 w-0.5 rounded-full bg-brand", className)} />
);
