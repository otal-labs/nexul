import { cn } from "@/lib/utils";

// The tree's columns: the row's words, then runs in 7 days, last run (from 1024px) and the switch.
export const treeGridClass =
  "grid grid-cols-[minmax(0,1fr)_3rem] items-center gap-x-3 sm:grid-cols-[minmax(0,1fr)_4.5rem_3rem] lg:grid-cols-[minmax(0,1fr)_4.5rem_5rem_3rem]";

export const treeRowClass = cn(treeGridClass, "relative cursor-pointer border-b border-border pr-3 hover:bg-accent");

// Opens the row from anywhere on it, except the switch, which saves at once.
export const stopRowClick = (event: { stopPropagation: () => void }) => event.stopPropagation();
