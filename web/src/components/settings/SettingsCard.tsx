import type { ComponentType, ReactNode } from "react";

import { SettingsHint } from "@/components/settings/SettingsHint";
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
  /** Shrinks to the height it is given instead of growing past it, so a scroller in its body takes the overflow. */
  fill?: boolean;
  /** Keeps the footer at the bottom of the view while a tall body scrolls under it, so its Save stays in reach. */
  stickyFooter?: boolean;
}

// The title and description open the card with no rule under them; the body follows, then the footer strip.
export const SettingsCard = ({
  id,
  title,
  description,
  danger = false,
  icon: Icon,
  children,
  footer,
  aside,
  fill = false,
  stickyFooter = false,
}: SettingsCardProps) => {
  const heading = (
    <div className={cn("flex min-w-0 flex-1 gap-3", Icon && !description ? "items-center" : "items-start")}>
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
        <h2 id={`${id}-title`} className="text-base font-semibold tracking-tight text-balance text-foreground">
          {title}
        </h2>
        {description && <SettingsHint className="mt-1 max-w-prose">{description}</SettingsHint>}
      </div>
      {/* Beside a lone title the aside centres on it; beside a title and description it keeps the top line. */}
      {aside && <div className={cn("flex shrink-0 items-center gap-2", !description && "self-center")}>{aside}</div>}
    </div>
  );

  return (
    <section
      id={id}
      aria-labelledby={`${id}-title`}
      className={cn(
        "scroll-mt-6 rounded-lg border bg-card shadow-card",
        danger ? "border-destructive/30" : "border-border",
        fill && "flex min-h-0 flex-col",
        // Clip, not hide: the card rounds a stuck footer's corners at rest without becoming the footer's scroll container.
        stickyFooter && "overflow-clip",
      )}
    >
      <header className="shrink-0 px-6 pt-5">{heading}</header>
      <div className={cn("@container p-6 pt-5", fill && "flex min-h-0 flex-col")}>{children}</div>
      {footer && (
        <footer
          className={cn(
            "flex shrink-0 flex-wrap items-center justify-between gap-3 rounded-b-lg border-t border-border bg-muted/30 px-6 py-3 empty:hidden",
            stickyFooter &&
              "sticky bottom-0 z-10 rounded-b-none bg-[color-mix(in_oklab,var(--muted)_30%,var(--card))] before:pointer-events-none before:absolute before:inset-x-0 before:-top-[calc(1.5rem+1px)] before:h-6 before:bg-linear-to-b before:from-transparent before:to-card",
          )}
        >
          {footer}
        </footer>
      )}
    </section>
  );
};
