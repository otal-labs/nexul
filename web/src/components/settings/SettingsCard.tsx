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
  /** The card's state or one quiet action, top right beside the title (Connected, Up to date). */
  aside?: ReactNode;
}

// The title and description open the card with no rule under them; the body follows, then the footer strip.
export const SettingsCard = ({ id, title, description, danger = false, icon: Icon, children, footer, aside }: SettingsCardProps) => {
  const heading = (
    <div className="flex min-w-0 flex-1 items-start gap-3">
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
      <div className="min-w-0 flex-1">
        <h2 id={`${id}-title`} className="text-[15px] font-semibold tracking-tight text-balance">
          {title}
        </h2>
        {description && <p className="mt-1 max-w-prose text-sm text-pretty text-muted-foreground">{description}</p>}
      </div>
      {aside && <div className="flex shrink-0 items-center gap-2">{aside}</div>}
    </div>
  );

  return (
    <section
      id={id}
      aria-labelledby={`${id}-title`}
      className={cn("scroll-mt-6 rounded-lg border bg-card shadow-card", danger ? "border-destructive/30" : "border-border")}
    >
      <header className="px-6 pt-5">{heading}</header>
      <div className="@container p-6 pt-5">{children}</div>
      {footer && (
        <footer className="flex flex-wrap items-center justify-between gap-3 rounded-b-lg border-t border-border bg-muted/30 px-6 py-3">
          {footer}
        </footer>
      )}
    </section>
  );
};
