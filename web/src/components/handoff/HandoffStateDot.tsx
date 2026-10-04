import type { HandoffState } from "@/models/Handoff";

import { cn } from "@/lib/utils";

const dotClass: Record<HandoffState, string> = {
  running: "bg-warning animate-[status-pulse_2.4s_ease-standard_infinite]",
  done: "bg-success",
  failed: "bg-destructive",
  interrupted: "bg-muted-foreground",
  left_running: "bg-info",
};

interface HandoffStateDotProps {
  state: HandoffState;
}

// Decorative (aria-hidden): the pill and the dialog say the state in words beside it.
export const HandoffStateDot = ({ state }: HandoffStateDotProps) => (
  <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full transition-colors duration-150 ease-standard", dotClass[state])} />
);
