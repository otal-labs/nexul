import type { ComponentType, ReactNode } from "react";

import { cn } from "@/lib/utils";

interface SettingsCardProps {
  id: string;
  title: string;
  description?: ReactNode;
  danger?: boolean;
  icon?: ComponentType<{ className?: string }>;
  children: ReactNode;
  /** Action strip under the body (a Save, an Add, a Rollback): the action lives with the section, not floating in the body. */
  footer?: ReactNode;
}

// `danger` swaps to the destructive border tone; `icon` is optional (only the danger zone uses it).
export const SettingsCard = ({
  id,
  title,
  description,
  danger = false,
  icon: Icon,
  children,
  footer,
}: SettingsCardProps) => (
  <section
    id={id}
    aria-labelledby={`${id}-title`}
    className={cn(
      "animate-in fade-in-0 slide-in-from-bottom-1 scroll-mt-6 rounded-lg border bg-card shadow-card duration-200 ease-out",
      danger ? "border-destructive/30" : "border-border",
    )}
  >
    <header className="flex items-start gap-3 border-b p-6 pb-5">
      {Icon && (
        <span
          className={cn(
            "flex size-8 shrink-0 items-center justify-center rounded-md border",
            danger ? "border-destructive/20 bg-destructive/10 text-destructive" : "border-border bg-muted/60 text-primary",
          )}
        >
          <Icon className="size-4" aria-hidden />
        </span>
      )}
      <div className="min-w-0">
        <h2 id={`${id}-title`} className="text-lg font-semibold tracking-tight">
          {title}
        </h2>
        {description && (
          <p className="mt-1.5 text-sm text-muted-foreground">{description}</p>
        )}
      </div>
    </header>
    <div className="p-6">{children}</div>
    {footer && (
      <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-border bg-muted/30 px-6 py-3">
        {footer}
      </footer>
    )}
  </section>
);
