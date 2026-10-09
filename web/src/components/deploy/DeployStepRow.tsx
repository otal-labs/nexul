import { useLayoutEffect, useRef } from "react";
import { CheckIcon, Loader2, XIcon } from "lucide-react";

import { playStep } from "@/components/deploy/stepMotion";

import { cn } from "@/lib/utils";
import { formatStepDuration, type DeployStep } from "@/utils/DeployLogUtility";

interface DeployStepRowProps {
  step: Pick<DeployStep, "label" | "state" | "durationMs">;
}

const node: Record<DeployStep["state"], string> = {
  done: "bg-success text-background",
  active: "bg-card text-info ring-1 ring-info",
  failed: "bg-destructive text-background",
  pending: "bg-card ring-1 ring-border",
  skipped: "bg-card ring-1 ring-border ring-dashed",
};

// One rung of the timeline: a node on a rail that runs down to the next step, solid once this step is done.
export const DeployStepRow = ({ step }: DeployStepRowProps) => {
  const muted = step.state === "pending" || step.state === "skipped";
  const row = useRef<HTMLLIElement>(null);
  const shown = useRef(step.state);
  useLayoutEffect(() => {
    playStep(row.current, shown.current, step.state);
    shown.current = step.state;
  }, [step.state]);
  return (
    <li ref={row} className="group/step relative flex items-start gap-3 pb-5 last:pb-0">
      <span
        aria-hidden
        data-step-rail
        className={cn(
          "absolute top-6 bottom-1 left-[9px] w-px origin-top group-last/step:hidden",
          step.state === "done" ? "bg-success/50" : "bg-border",
        )}
      />
      <span data-step-node className={cn("relative mt-0.5 flex size-[19px] shrink-0 items-center justify-center rounded-full", node[step.state])} aria-hidden>
        {step.state === "done" && <CheckIcon className="size-3" strokeWidth={3} />}
        {step.state === "active" && <Loader2 className="size-3 animate-spin motion-reduce:animate-none" />}
        {step.state === "failed" && <XIcon className="size-3" strokeWidth={3} />}
      </span>
      <span className={cn("min-w-0 flex-1 text-sm leading-6", muted && "text-muted-foreground", step.state === "active" && "font-medium")}>
        {step.label}
      </span>
      <span className="sr-only">{step.state}</span>
      <span className="shrink-0 font-mono text-xs leading-6 text-muted-foreground tabular-nums">{formatStepDuration(step.durationMs)}</span>
    </li>
  );
};
