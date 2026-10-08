import type { ReactNode } from "react";

import { PageBreadcrumb, type Crumb } from "@/components/PageBreadcrumb";
import { cn } from "@/lib/utils";

export const pageTitleClass = "text-2xl font-semibold tracking-tight text-balance break-words";

interface PageHeaderProps {
  title: ReactNode;
  crumbs?: Crumb[];
  meta?: ReactNode;
  actions?: ReactNode;
  className?: string;
}

export const PageHeader = ({ title, crumbs, meta, actions, className }: PageHeaderProps) => (
  <header className={cn("border-b border-border pb-5", className)}>
    {crumbs && crumbs.length > 0 && <PageBreadcrumb crumbs={crumbs} className="mb-2" />}
    <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-3">
      <div className="min-w-0 flex-1 basis-80">
        {typeof title === "string" && <h1 className={pageTitleClass}>{title}</h1>}
        {typeof title !== "string" && title}
        {meta && (
          <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
            {meta}
          </div>
        )}
      </div>
      {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
    </div>
  </header>
);
