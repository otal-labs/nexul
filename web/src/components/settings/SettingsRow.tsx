import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface SettingsRowProps {
  label: ReactNode;
  description?: ReactNode;
  /** Makes the label a <label> for the control's id, so a click on it focuses the field. */
  htmlFor?: string;
  /** The control, right of the text from a 34rem card and under it below that. */
  children?: ReactNode;
  /** Puts the control under the text at every width, for a control that needs the room (a list, a textarea). */
  stacked?: boolean;
  className?: string;
}

// One setting: what it is and what it does on the left, the control on the right. Rows sit in SettingsRows.
export const SettingsRow = ({ label, description, htmlFor, children, stacked = false, className }: SettingsRowProps) => {
  const Label = htmlFor ? "label" : "p";
  return (
    <div
      className={cn(
        "flex flex-col gap-3 py-4 first:pt-0 last:pb-0",
        !stacked && "@[34rem]:flex-row @[34rem]:items-center @[34rem]:justify-between @[34rem]:gap-8",
        className,
      )}
    >
      <div className={cn("min-w-0 space-y-1", !stacked && "@[34rem]:max-w-[24rem]")}>
        <Label {...(htmlFor ? { htmlFor } : {})} className="block text-sm font-medium text-pretty">
          {label}
        </Label>
        {description && <div className="text-sm text-pretty text-muted-foreground">{description}</div>}
      </div>
      {children && (
        <div className={cn("min-w-0", !stacked && "@[34rem]:flex @[34rem]:shrink-0 @[34rem]:justify-end @[34rem]:w-[min(20rem,50%)]")}>
          {children}
        </div>
      )}
    </div>
  );
};

interface SettingsRowsProps {
  children: ReactNode;
  className?: string;
}

// Rows split by hairlines; the container query lets a row put its control beside the text only when the card has room.
export const SettingsRows = ({ children, className }: SettingsRowsProps) => (
  <div className={cn("@container divide-y divide-border", className)}>{children}</div>
);
