import { CheckIcon, Loader2, XIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import { formatStepDuration, type DeployStep } from "@/utils/DeployLogUtility";

interface DeployStepRowProps {
  step: DeployStep;
}

export const DeployStepRow = ({ step }: DeployStepRowProps) => {
  const muted = step.state === "pending" || step.state === "skipped";
  return (
    <li className="flex items-center gap-3 py-3">
      <span className="flex size-4 shrink-0 items-center justify-center" aria-hidden>
        {step.state === "done" && <CheckIcon className="size-4 text-success" />}
        {step.state === "active" && <Loader2 className="size-4 animate-spin text-muted-foreground motion-reduce:animate-none" />}
        {step.state === "failed" && <XIcon className="size-4 text-destructive" />}
      </span>
      <span className={cn("min-w-0 flex-1 truncate text-sm", muted && "text-muted-foreground")}>{step.label}</span>
      <span className="sr-only">{step.state}</span>
      <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">{formatStepDuration(step.durationMs)}</span>
    </li>
  );
};
