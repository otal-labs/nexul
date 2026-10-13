import { Check } from "lucide-react";
import { createContext, useContext, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { useWelcomeProto, type StepKey } from "./model";

export interface Flow {
  step: StepKey;
  first: boolean;
  last: boolean;
  back: () => void;
  // Records the step and moves on; the last step moves on to the finish view.
  next: (as: "done" | "skipped") => void;
  // The hub keeps Back out of its rows, and its rows put the footer flush left.
  compact?: boolean;
}

export const FlowContext = createContext<Flow | null>(null);
export const useFlow = () => {
  const flow = useContext(FlowContext);
  if (!flow) throw new Error("no flow");
  return flow;
};

interface StepFooterProps {
  primary?: { label: string; disabled?: boolean; form?: string; onClick?: () => void; icon?: ReactNode } | undefined;
  // Hidden once the step is done: there is nothing left to skip.
  skip?: boolean;
}

// The project wizard's footer: Back on the left, then a ghost Skip for now, then the one primary.
export const StepFooter = ({ primary, skip = true }: StepFooterProps) => {
  const flow = useFlow();
  return (
    <div className={cn("flex items-center gap-2 pt-8", flow.compact && "pt-5")}>
      {!flow.compact && !flow.first && (
        <Button variant="ghost" className="-ml-3 text-muted-foreground" onClick={flow.back}>
          Back
        </Button>
      )}
      <div className={cn("flex items-center gap-2", !flow.compact && "ml-auto")}>
        {skip && (
          <Button variant="ghost" className="text-muted-foreground" onClick={() => flow.next("skipped")}>
            Skip for now
          </Button>
        )}
        {primary && (
          <Button
            type={primary.form ? "submit" : "button"}
            form={primary.form}
            disabled={primary.disabled}
            onClick={primary.form ? undefined : (primary.onClick ?? (() => flow.next("done")))}
          >
            {primary.icon}
            {primary.label}
          </Button>
        )}
      </div>
    </div>
  );
};

// A step that was already done when the wizard reached it says what it found, and the footer only moves on.
export const AlreadyDone = ({ children }: { children: ReactNode }) => (
  <div className="flex items-start gap-3 rounded-lg border border-border bg-card px-4 py-3">
    <span className="mt-0.5 grid size-5 shrink-0 place-items-center rounded-full border border-border text-success">
      <Check className="pop-in size-3" strokeWidth={3} aria-hidden />
    </span>
    <div className="min-w-0 space-y-0.5 text-sm">{children}</div>
  </div>
);

export const useStepState = (key: StepKey) => useWelcomeProto((s) => s.states[key]);
