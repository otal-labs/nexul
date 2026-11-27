import type { ReactNode } from "react";

import { displayTitleClass } from "@/components/PageHeader";

interface PaneHeaderProps {
  title: string;
  /** One muted line under the title: a state, or what the settings are for. */
  meta?: ReactNode;
  actions?: ReactNode;
}

// The head of the Computers dialog's right pane: the display-face name over one line, its actions at the right.
export const PaneHeader = ({ title, meta, actions }: PaneHeaderProps) => (
  <header className="flex flex-wrap items-start justify-between gap-x-4 gap-y-3">
    <div className="min-w-0 space-y-1">
      <h3 dir="auto" className={`${displayTitleClass} text-xl`}>
        {title}
      </h3>
      {meta && <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-muted-foreground">{meta}</div>}
    </div>
    {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
  </header>
);
