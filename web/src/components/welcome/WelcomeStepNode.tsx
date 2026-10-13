import { Check, Minus } from "lucide-react";

import type { StepMark } from "@/models/WelcomeWizard";
import { cn } from "@/lib/utils";

interface WelcomeStepNodeProps {
  state: StepMark | "current";
}

// The project wizard's node: a success check, a dashed ring with a dash, a hollow ring, or the brand ring with its dot.
export const WelcomeStepNode = ({ state }: WelcomeStepNodeProps) => (
  <span
    aria-hidden
    className={cn(
      "relative z-10 flex size-5 shrink-0 items-center justify-center rounded-full bg-background",
      state === "current" && "border-2 border-brand",
      state === "done" && "border border-border text-success",
      state === "skipped" && "border border-dashed border-muted-foreground/60 text-muted-foreground",
      state === "left" && "border border-muted-foreground/40",
    )}
  >
    {state === "current" && <span className="size-2 rounded-full bg-brand" />}
    {state === "done" && <Check className="pop-in size-3" strokeWidth={3} />}
    {state === "skipped" && <Minus className="size-3" strokeWidth={3} />}
  </span>
);
