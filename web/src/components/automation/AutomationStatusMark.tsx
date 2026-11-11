import { Circle } from "lucide-react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { Automation } from "@/models/Automation";
import { runOutcomes } from "@/models/AutomationRun";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface AutomationStatusMarkProps {
  automation: Automation;
}

// The last run's outcome in the row's leading slot: a hollow ring before the first run, muted while the automation is off.
export const AutomationStatusMark = ({ automation }: AutomationStatusMarkProps) => {
  const run = automation.last_run;
  const mark = run ? runOutcomes[run.outcome] : { icon: Circle, color: "text-muted-foreground/60", past: "" };
  const label = run ? `Last run ${mark.past} ${formatRelativeTime(run.started_at)}` : "Never ran";
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
