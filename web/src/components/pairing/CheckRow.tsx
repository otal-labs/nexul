import type { LucideIcon } from "lucide-react";

import type { Check, CheckState } from "@/models/ComputerChecks";
import { cn } from "@/lib/utils";

// Status is the only colour on the dialog; waiting stays monochrome and pulses.
const STATE_DOT: Record<CheckState, string> = {
  waiting: "bg-muted-foreground/60 animate-[status-pulse_2.4s_ease-standard_infinite]",
  passed: "bg-success",
  warning: "bg-warning",
  failed: "bg-destructive",
};

interface CheckRowProps {
  icon: LucideIcon;
  check: Check;
}

// One live check: an icon tile, the check's name over a mono line of what it found, and its state as a dot and a word.
export const CheckRow = ({ icon: Icon, check }: CheckRowProps) => (
  <li className="flex items-center gap-2.5 px-3 py-2.5">
    <span className="grid size-7 shrink-0 place-items-center rounded-md border border-border bg-background text-muted-foreground">
      <Icon className="size-3.5" aria-hidden />
    </span>
    <span className="flex min-w-0 flex-1 flex-col gap-0.5">
      <span className="text-xs font-medium break-words">{check.name}</span>
      <span className="font-mono text-xs break-words text-muted-foreground">{check.detail}</span>
    </span>
    <span className="flex shrink-0 items-center gap-1.5 text-xs font-medium text-muted-foreground">
      {/* Keyed by state so a check that passes pops its dot in, the way a finished deploy step lands. */}
      <span
        key={check.state}
        className={cn("size-1.5 rounded-full transition-colors duration-150 ease-standard", STATE_DOT[check.state], check.state === "passed" && "pop-in")}
        aria-hidden
      />
      {check.label}
    </span>
  </li>
);
