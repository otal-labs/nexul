import { cn } from "@/lib/utils";

// Every property row shares this shape; editable rows add the hover affordance, read-only rows don't.
export const rowClass = "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left";
export const editableRowClass = cn(rowClass, "cursor-default transition-colors duration-150 ease-standard hover:bg-muted/50");
export const menuItemClass =
  "flex items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs text-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 focus-visible:bg-accent/60";
