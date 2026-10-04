import type { ReactNode } from "react";
import { ChevronDownIcon } from "lucide-react";

import { cn } from "@/lib/utils";

interface InterviewSectionProps {
  label: string;
  meta: string;
  folded: boolean;
  onToggle: () => void;
  // A control trailing the header, outside the fold button.
  action?: ReactNode;
  children: ReactNode;
}

// A foldable group of question rows with the board swimlane's header: label left, muted count and chevron right; an
// action that doesn't fit beside the whole count wraps to its own line rather than cutting the count short.
export const InterviewSection = ({ label, meta, folded, onToggle, action, children }: InterviewSectionProps) => (
  <section aria-label={label} className="space-y-2">
    <h3 className="flex flex-wrap items-center justify-end gap-x-2 gap-y-1 border-b border-border pb-1">
      <button
        type="button"
        aria-expanded={!folded}
        onClick={onToggle}
        className="group/lane flex grow cursor-pointer items-center justify-between gap-2 rounded-sm px-1.5 py-1 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
      >
        <span className="shrink-0 text-sm font-semibold">{label}</span>
        <span className="flex shrink-0 items-center gap-1 font-mono text-xs text-muted-foreground tabular-nums transition-colors group-hover/lane:text-foreground">
          {meta}
          <ChevronDownIcon
            className={cn("size-3.5 shrink-0 transition-transform duration-150 ease-standard", folded && "-rotate-90")}
            aria-hidden
          />
        </span>
      </button>
      {action}
    </h3>
    <div
      inert={folded}
      aria-hidden={folded}
      className={cn(
        "grid transition-[grid-template-rows,opacity] motion-reduce:transition-[opacity]",
        folded
          ? "[grid-template-rows:0fr] opacity-0 duration-150 ease-standard"
          : "[grid-template-rows:1fr] opacity-100 duration-200 ease-out",
      )}
    >
      <div className="min-h-0 overflow-hidden">{children}</div>
    </div>
  </section>
);
