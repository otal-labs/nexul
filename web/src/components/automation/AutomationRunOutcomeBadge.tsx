import { NoFillBadge } from "@/components/ui/badge";
import type { AutomationRunOutcome } from "@/enums/Automation";
import { runOutcomes } from "@/models/AutomationRun";

interface AutomationRunOutcomeBadgeProps {
  outcome: AutomationRunOutcome;
}

export const AutomationRunOutcomeBadge = ({ outcome }: AutomationRunOutcomeBadgeProps) => {
  const mark = runOutcomes[outcome];
  return (
    <NoFillBadge icon={mark.icon} color={mark.color}>
      {mark.word}
    </NoFillBadge>
  );
};
