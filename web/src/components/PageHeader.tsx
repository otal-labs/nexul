import type { ReactNode } from "react";

import { PageBreadcrumb, type Crumb } from "@/components/PageBreadcrumb";
import { cn } from "@/lib/utils";

// The display face for a headline; never below 20px. Empty-state and showcase headlines add their own size.
export const displayTitleClass = "type-display text-balance break-words";

export const pageTitleClass = `${displayTitleClass} text-[1.75rem]`;

interface PageHeaderProps {
  title: ReactNode;
  crumbs?: Crumb[];
  meta?: ReactNode;
  actions?: ReactNode;
  // A mark before the title and its meta line, centred on the pair: the board's project mark.
  leading?: ReactNode;
  className?: string;
}

export const PageHeader = ({ title, crumbs, meta, actions, leading, className }: PageHeaderProps) => (
  <header className={cn("border-b border-border pb-5", className)}>
    {crumbs && crumbs.length > 0 && <PageBreadcrumb crumbs={crumbs} className="mb-2" />}
    <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-3">
      <div className="flex min-w-0 flex-1 basis-80 items-center gap-3.5">
        {leading}
        <div className="min-w-0 flex-1">
          {typeof title === "string" && <h1 className={pageTitleClass}>{title}</h1>}
          {typeof title !== "string" && title}
          {meta && (
            <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
              {meta}
            </div>
          )}
        </div>
      </div>
      {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
    </div>
  </header>
);
