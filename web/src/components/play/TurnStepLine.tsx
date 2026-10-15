import { isFailedStep, stepPhrase } from "@nexul/client-core/agentSteps";

import { StepDetailDisclosure } from "@/components/play/StepDetailDisclosure";
import { StepStatusMark } from "@/components/play/StepStatusMark";
import type { ActivityEntry } from "@/models/Trail";

interface TurnStepLineProps {
  entry: ActivityEntry;
  live: boolean;
}

// Any other tool as its verb and name, opening to its detail; the Agent's own sentence reads whole as plain text.
export const TurnStepLine = ({ entry, live }: TurnStepLineProps) => {
  const { verb, object } = stepPhrase(entry, live);
  const said = verb === "";
  const line = (
    <span className="flex items-center gap-1.5 text-sm text-foreground/80">
      <span className="shrink-0">{verb}</span>
      <span className="min-w-0 truncate font-mono text-xs text-muted-foreground">{object}</span>
      <StepStatusMark live={live} failed={isFailedStep(entry)} />
    </span>
  );
  return (
    <div>
      {said && <p className="text-sm break-words whitespace-pre-wrap text-foreground/80">{entry.kind === "text" && entry.detail ? entry.detail : object}</p>}
      {!said && !entry.detail && line}
      {!said && entry.detail && (
        <StepDetailDisclosure detail={entry.detail} at={entry.at}>
          {line}
        </StepDetailDisclosure>
      )}
    </div>
  );
};
