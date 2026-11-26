import type { ReactNode } from "react";

interface AutomationListBarProps {
  summary: ReactNode;
  children?: ReactNode;
}

// The line over a list on the Automations pages: what the list holds at the left, its one control at the right.
export const AutomationListBar = ({ summary, children }: AutomationListBarProps) => (
  <div className="flex min-h-8 flex-wrap items-center justify-between gap-x-3 gap-y-2 px-4">
    <p className="min-w-0 text-xs text-muted-foreground">{summary}</p>
    {children}
  </div>
);
