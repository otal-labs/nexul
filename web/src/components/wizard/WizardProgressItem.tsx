import { Check, Minus } from "lucide-react";

import { cn } from "@/lib/utils";

export type WizardProgressState = "done" | "skipped" | "current" | "unvisited";

interface WizardProgressItemProps {
  label: string;
  state: WizardProgressState;
  first: boolean;
  // Whether the connector into this node fills: every node up to the one on screen.
  reached: boolean;
  onSelect: () => void;
}

const node = "relative z-10 flex size-5 items-center justify-center rounded-full bg-panel";
const control = "outline-none focus-visible:ring-[3px] focus-visible:ring-ring/30 hover:border-foreground";

const spoken: Record<Exclude<WizardProgressState, "current">, (label: string) => string> = {
  done: (label) => `${label}, done`,
  skipped: (label) => `${label}, skipped`,
  unvisited: (label) => label,
};

// One node on the row: every step but the one on screen is a button, whatever happened there, so any step can be opened.
export const WizardProgressItem = ({ label, state, first, reached, onSelect }: WizardProgressItemProps) => (
  <li className="relative flex flex-1 flex-col items-center" data-state={state}>
    {!first && (
      <span aria-hidden className="absolute top-2.5 right-1/2 h-px w-full bg-border">
        <span
          className={cn(
            "block h-full origin-left bg-brand transition-transform duration-250 ease-standard grow-in",
            reached ? "scale-x-100" : "scale-x-0",
          )}
        />
      </span>
    )}
    {state === "current" && (
      <span aria-current="step" className={cn(node, "border-2 border-brand")}>
        <span aria-hidden className="size-2 rounded-full bg-brand" />
        <span className="sr-only">{label}</span>
      </span>
    )}
    {state !== "current" && (
      <button
        type="button"
        onClick={onSelect}
        className={cn(
          node,
          control,
          state === "done" && "border border-border text-success",
          state === "skipped" && "border border-dashed border-muted-foreground/60 text-muted-foreground",
          state === "unvisited" && "border border-muted-foreground/40",
        )}
      >
        {state === "done" && <Check className="size-3" strokeWidth={3} aria-hidden />}
        {state === "skipped" && <Minus className="size-3" strokeWidth={3} aria-hidden />}
        <span className="sr-only">{spoken[state](label)}</span>
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
