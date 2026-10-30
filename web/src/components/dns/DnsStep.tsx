import { Check } from "lucide-react";
import type { ReactNode } from "react";

import { SettingsHint } from "@/components/settings/SettingsHint";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export type DnsStepState = "upcoming" | "active" | "done";

interface DnsStepProps {
  title: string;
  description?: string | undefined;
  state: DnsStepState;
  // Replaces the body once done; Change reopens the step.
  summary?: ReactNode;
  onChange?: () => void;
  last?: boolean;
  children?: ReactNode;
}

// One rung of the setup stepper (Resend "Add domain" lock): rail on the left, content straight on the canvas, no card.
export const DnsStep = ({ title, description, state, summary, onChange, last = false, children }: DnsStepProps) => (
  <li className="relative flex gap-4 pb-8 last:pb-0" data-state={state}>
    <div className="flex flex-col items-center">
      <span
        aria-hidden
        className={cn(
          "mt-1 flex size-4 shrink-0 items-center justify-center rounded-full border transition-colors duration-250 ease-standard",
          state === "upcoming" && "border-border",
          state === "active" && "border-primary bg-primary",
          state === "done" && "border-primary text-primary",
        )}
      >
        {state === "done" && (
          <Check className="size-2.5 animate-in zoom-in-50 duration-150 ease-out" strokeWidth={3} />
        )}
      </span>
      {!last && <span aria-hidden className="dns-step-rail mt-1 -mb-8 w-px flex-1 bg-border" />}
    </div>
    <div className="min-w-0 flex-1">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h2
            className={cn(
              "text-base font-semibold tracking-tight transition-colors duration-250 ease-standard",
              state === "upcoming" && "text-muted-foreground",
            )}
          >
            {title}
          </h2>
          {description && state === "active" && (
            <SettingsHint className="mt-1">{description}</SettingsHint>
          )}
        </div>
        {state === "done" && onChange && (
          <Button variant="ghost" size="sm" className="-mt-1 -mr-3 text-muted-foreground" onClick={onChange}>
            Change
          </Button>
        )}
      </div>
      {state === "done" && summary && (
        <SettingsHint className="dns-step-summary mt-1">{summary}</SettingsHint>
      )}
      {state === "active" && children && (
        <div className="dns-step-body mt-4">
          <div>{children}</div>
        </div>
      )}
    </div>
  </li>
);
