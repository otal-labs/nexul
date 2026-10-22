import { ChevronDownIcon } from "lucide-react";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface RunSettingsRowProps {
  mark: ReactNode;
  title: string;
  // One muted line of what its runs use; undefined while it loads.
  summary: string | undefined;
  open: boolean;
  onToggle: () => void;
  children: ReactNode;
}

// One row of the Projects list, the same for Defaults and for every project: a mark, the name over what its runs use,
// and a chevron folding its editor open in a well beneath it.
export const RunSettingsRow = ({ mark, title, summary, open, onToggle, children }: RunSettingsRowProps) => (
  <li>
    <button
      type="button"
      aria-expanded={open}
      onClick={onToggle}
      className="group flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors duration-150 ease-standard hover:bg-accent/40 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-focus"
    >
      {mark}
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium" title={title}>
          {title}
        </span>
        {summary === undefined && <span aria-hidden className="mt-1 block h-3 w-32 rounded-sm bg-muted" />}
        {summary !== undefined && <span className="block truncate text-xs text-muted-foreground">{summary}</span>}
      </span>
      <ChevronDownIcon
        aria-hidden
        className={cn("size-4 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard group-hover:text-foreground", open && "rotate-180")}
      />
    </button>
    {open && (
      <div className="settle-in border-t border-border bg-surface-2 px-4 py-3">
        <div className="@container max-w-2xl">{children}</div>
      </div>
    )}
  </li>
);
