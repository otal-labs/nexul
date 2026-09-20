import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  eyebrow?: string;
  actions?: ReactNode;
  className?: string;
}

export const PageHeader = ({ title, subtitle, eyebrow, actions, className }: PageHeaderProps) => (
  <header className={cn("flex flex-wrap items-end justify-between gap-3", className)}>
    <div className="min-w-0 space-y-1">
      {eyebrow && (
        <p className="font-mono text-[11px] font-medium tracking-[0.18em] text-primary/80 uppercase">
          {eyebrow}
        </p>
      )}
      <h1 className="text-2xl font-semibold tracking-tight sm:text-[28px]">{title}</h1>
      {subtitle && <p className="max-w-2xl text-sm text-muted-foreground">{subtitle}</p>}
    </div>
    {actions && <div className="flex items-center gap-2">{actions}</div>}
  </header>
);
