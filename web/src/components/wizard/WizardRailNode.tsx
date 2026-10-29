import { Check, Minus } from "lucide-react";

import type { WizardRailMark } from "@/models/WizardRail";
import { cn } from "@/lib/utils";

interface WizardRailNodeProps {
  mark: WizardRailMark;
}

// A success check, a dashed ring with a dash, a hollow ring, or the brand ring with its dot.
export const WizardRailNode = ({ mark }: WizardRailNodeProps) => (
  <span
    aria-hidden
    className={cn(
      "relative z-10 flex size-5 shrink-0 items-center justify-center rounded-full",
      mark === "current" && "border-2 border-brand",
      mark === "done" && "border border-border text-success",
      (mark === "skipped" || mark === "later") && "border border-dashed border-muted-foreground/60 text-muted-foreground",
      mark === "left" && "border border-muted-foreground/60 text-muted-foreground",
      mark === "locked" && "border border-muted-foreground/40",
    )}
  >
    {mark === "current" && <span className="size-2 rounded-full bg-brand" />}
    {mark === "done" && <Check className="pop-in size-3" strokeWidth={3} />}
    {(mark === "skipped" || mark === "later") && <Minus className="size-3" strokeWidth={3} />}
  </span>
);
