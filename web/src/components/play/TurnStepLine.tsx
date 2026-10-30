import { isFailedStep, stepAct, stepPhrase } from "@nexul/client-core/agentSteps";

import { StepDetailDisclosure } from "@/components/play/StepDetailDisclosure";
import { StepStatusMark } from "@/components/play/StepStatusMark";
import { TurnStepIcon } from "@/components/play/TurnStepIcon";
import type { ActivityEntry } from "@/models/Trail";

interface TurnStepLineProps {
  entry: ActivityEntry;
  live: boolean;
  root: string | null;
}

// A lone command or any other tool as one row, its icon, verb and mono name, opening to its detail; the Agent's own
// sentence reads whole as plain text.
export const TurnStepLine = ({ entry, live, root }: TurnStepLineProps) => {
  const { verb, object } = stepPhrase(entry, live, root);
  const said = verb === "";
  const line = (
    <span className="flex min-w-0 items-center gap-2 text-[13px] leading-5">
      <TurnStepIcon act={stepAct(entry)} tool={entry.tool} />
      <span className="shrink-0 text-muted-foreground">{verb}</span>
      <span className="min-w-0 truncate font-mono text-xs leading-5 text-foreground/80" title={object}>
        {object}
      </span>
      <StepStatusMark live={live} failed={isFailedStep(entry)} />
    </span>
  );
  return (
    <div>
      {said && <p className="text-sm break-words whitespace-pre-wrap text-foreground/85">{entry.kind === "text" && entry.detail ? entry.detail : object}</p>}
      {!said && !entry.detail && line}
      {!said && entry.detail && (
        <StepDetailDisclosure detail={entry.detail} at={entry.at}>
          {line}
        </StepDetailDisclosure>
      )}
    </div>
  );
};
