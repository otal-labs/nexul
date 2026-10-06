import { cn } from "@/lib/utils";

// Every property row shares this shape; editable rows add the hover affordance, read-only rows don't.
export const rowClass = "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left";
export const editableRowClass = cn(rowClass, "cursor-default transition-colors duration-150 ease-standard hover:bg-muted/50");
// The leading icon slot and the label column keep every row's value starting at the same x.
export const rowIconClass = "flex size-4 shrink-0 items-center justify-center text-muted-foreground";
export const rowLabelClass = "w-16 shrink-0 text-xs text-muted-foreground";
export const rowValueClass = "min-w-0 truncate text-xs text-foreground";
export const menuItemClass =
  "flex items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs text-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 focus-visible:bg-accent/60";
