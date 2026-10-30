// Shared tracks, so a view's titles start on one edge; under 55rem (880px) of grid a month meeting's time gives way to a dot.
export const MONTH_ROW = "grid h-5 grid-cols-[2.5rem_minmax(0,1fr)] items-center gap-x-1 @max-[55rem]:grid-cols-[0.375rem_minmax(0,1fr)] @max-[55rem]:px-1";
export const WEEK_ROW = "grid h-6 grid-cols-[7.5rem_minmax(0,1fr)] items-center gap-x-3 @min-[68.75rem]:grid-cols-[7.5rem_minmax(0,1fr)_auto]";
export const ENTRY_ROW = "w-full min-w-0 rounded-md px-1.5 text-left transition-colors duration-150 ease-standard hover:bg-accent";
// A month's due ticket, at the label chip's fill, so untimed work reads apart from the meetings.
export const DUE_BOX =
  "grid h-[19px] grid-cols-[0.875rem_minmax(0,1fr)] items-center gap-x-1 rounded-sm bg-foreground/[0.04] px-1.5 text-xs ring-1 ring-border ring-inset transition-colors duration-150 ease-standard hover:bg-foreground/[0.1] @max-[55rem]:grid-cols-1 @max-[55rem]:px-1";
