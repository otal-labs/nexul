import { CheckCircle2, Skull, XCircle } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { AutomationRunOutcome } from "@/enums/Automation";

interface AutomationRunOutcomeBadgeProps {
  outcome: AutomationRunOutcome;
}

// Success/fail need to read as visually distinct at a glance (AM10) — bounded
// enum, so a colored icon leads per the Mono Console badge convention.
export const AutomationRunOutcomeBadge = ({ outcome }: AutomationRunOutcomeBadgeProps) => {
  if (outcome === AutomationRunOutcome.Success) {
    return (
      <NoFillBadge icon={CheckCircle2} color="text-success">
        Success
      </NoFillBadge>
    );
  }
  if (outcome === AutomationRunOutcome.Crash) {
    return (
      <NoFillBadge icon={Skull} color="text-destructive">
        Crash
      </NoFillBadge>
    );
  }
  return (
    <NoFillBadge icon={XCircle} color="text-destructive">
      Failure
    </NoFillBadge>
  );
};
