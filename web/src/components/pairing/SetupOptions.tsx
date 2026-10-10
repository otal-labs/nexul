import { useState, type ReactNode } from "react";
import { ChevronRight } from "lucide-react";

import { cn } from "@/lib/utils";

interface SetupOptionsProps {
  // Open while nothing has run yet, folded once the choices are made.
  defaultOpen: boolean;
  // The choices read back in one mono line, so a folded section still says what a run will use.
  summary: string;
  children: ReactNode;
}

// The run's providers, models, folder, and installs behind one disclosure under the run.
export const SetupOptions = ({ defaultOpen, summary, children }: SetupOptionsProps) => {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div>
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className="group -ml-1 flex w-full min-w-0 items-center gap-1.5 rounded-md px-1 py-0.5 text-left text-sm focus-visible:outline-2 focus-visible:outline-focus"
      >
        <ChevronRight
          aria-hidden
          className={cn("size-4 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard group-hover:text-foreground", open && "rotate-90")}
        />
        <span className="font-medium">Options</span>
        <span className="min-w-0 truncate font-mono text-xs text-muted-foreground">{summary}</span>
      </button>
      <div inert={!open} aria-hidden={!open} data-closed={!open || undefined} className="disclosure">
        <div>
          <div className="space-y-6 pt-4 pl-6">{children}</div>
        </div>
      </div>
    </div>
  );
};
