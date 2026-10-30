import { CheckCircle2, Circle, Skull, XCircle, type LucideIcon } from "lucide-react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { AutomationRunOutcome } from "@/enums/Automation";
import { cn } from "@/lib/utils";
import type { Automation } from "@/models/Automation";
import { formatRelativeTime } from "@/utils/TimeUtility";

const marks: Record<AutomationRunOutcome, { icon: LucideIcon; color: string; word: string }> = {
  [AutomationRunOutcome.Success]: { icon: CheckCircle2, color: "text-success", word: "succeeded" },
  [AutomationRunOutcome.Failure]: { icon: XCircle, color: "text-destructive", word: "failed" },
  [AutomationRunOutcome.Crash]: { icon: Skull, color: "text-destructive", word: "crashed" },
};

interface AutomationStatusMarkProps {
  automation: Automation;
}

// The last run's outcome in the row's leading slot: a hollow ring before the first run, muted while the automation is off.
export const AutomationStatusMark = ({ automation }: AutomationStatusMarkProps) => {
  const run = automation.last_run;
  const mark = run ? marks[run.outcome] : { icon: Circle, color: "text-muted-foreground/60", word: "" };
  const label = run ? `Last run ${mark.word} ${formatRelativeTime(run.started_at)}` : "Never ran";
  const Icon = mark.icon;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span tabIndex={0} aria-label={label} className="grid size-5 place-items-center rounded-sm">
          <Icon className={cn("size-4", mark.color, !automation.enabled && "opacity-50")} aria-hidden />
        </span>
      </TooltipTrigger>
      <TooltipContent side="top" align="start" className="max-w-sm">
        <p>{label}</p>
        {run?.error && <p className="mt-1 font-mono text-xs break-words opacity-80">{run.error}</p>}
      </TooltipContent>
    </Tooltip>
  );
};
