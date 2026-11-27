import { ChevronRightIcon } from "lucide-react";

import { cn } from "@/lib/utils";

// The cue that a row opens: a muted chevron that turns down while it is open.
export const RowChevron = ({ open, className }: { open: boolean; className?: string }) => (
  <ChevronRightIcon aria-hidden className={cn("size-4 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard motion-reduce:transition-none", open && "rotate-90", className)} />
);
