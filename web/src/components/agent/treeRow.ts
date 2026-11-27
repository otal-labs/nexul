import { cn } from "@/lib/utils";

// The tree's columns: the row's words, then 7 days, last run and the switch; a phone keeps the words and the switch.
export const treeGridClass = "grid grid-cols-[minmax(0,1fr)_2.75rem] items-center gap-x-3 sm:grid-cols-[minmax(0,1fr)_3.5rem_4.5rem_2.75rem]";

export const treeRowClass = cn(treeGridClass, "relative border-b border-border pr-3 hover:bg-accent");
