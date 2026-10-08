import { Check } from "lucide-react";

import { cn } from "@/lib/utils";

export type WizardProgressState = "done" | "current" | "future";

interface WizardProgressItemProps {
  label: string;
  state: WizardProgressState;
  first: boolean;
  // Only a done step that is safe to revisit gets one; without it the node is not a control.
  onSelect?: (() => void) | undefined;
}

const node = "relative z-10 flex size-5 items-center justify-center rounded-full bg-panel";
const focus = "outline-none focus-visible:ring-[3px] focus-visible:ring-ring/30";

// One node on the row: the connector comes in from the previous node and fills once this step is reached.
export const WizardProgressItem = ({ label, state, first, onSelect }: WizardProgressItemProps) => (
  <li className="relative flex flex-1 flex-col items-center" data-state={state}>
    {!first && (
      <span aria-hidden className="absolute top-2.5 right-1/2 h-px w-full bg-border">
        <span
          className={cn(
            "block h-full origin-left bg-brand transition-transform duration-200 ease-out motion-reduce:transition-none",
            state === "future" ? "scale-x-0" : "scale-x-100",
          )}
        />
      </span>
    )}
    {state === "done" && onSelect && (
      <button
        type="button"
        onClick={onSelect}
        className={cn(node, focus, "border border-border text-success hover:border-foreground")}
      >
        <Check className="size-3" strokeWidth={3} aria-hidden />
        <span className="sr-only">Back to {label}</span>
      </button>
    )}
    {state === "done" && !onSelect && (
      <span className={cn(node, "border border-border text-success")}>
        <Check className="size-3" strokeWidth={3} aria-hidden />
        <span className="sr-only">{label}, done</span>
      </span>
    )}
    {state === "current" && (
      <span aria-current="step" className={cn(node, "border-2 border-brand")}>
        <span aria-hidden className="size-2 rounded-full bg-brand" />
        <span className="sr-only">{label}</span>
      </span>
    )}
    {state === "future" && (
      <button type="button" disabled className={cn(node, "border border-muted-foreground/40")}>
        <span className="sr-only">{label}</span>
      </button>
    )}
    <span
      aria-hidden
      className={cn(
        "mt-2 hidden px-1 text-center font-mono text-xs leading-tight @2xl:block",
        state === "current" ? "font-bold text-foreground" : "text-muted-foreground",
      )}
    >
      {label}
    </span>
  </li>
);
