import { isFailedStep, stepPhrase } from "@nexul/client-core/agentSteps";

import { StepDetailDisclosure } from "@/components/play/StepDetailDisclosure";
import { StepStatusMark } from "@/components/play/StepStatusMark";
import type { ActivityEntry } from "@/models/Trail";

interface TurnCommandStepProps {
  entry: ActivityEntry;
  live: boolean;
}

// A command as one line saying it ran, over the command itself in a well, which opens to its output.
export const TurnCommandStep = ({ entry, live }: TurnCommandStepProps) => {
  const { verb, object } = stepPhrase(entry, live);
  const failed = isFailedStep(entry);
  const well = (
    <span className="mt-1 flex items-start gap-2 rounded-md bg-surface-2 px-2 py-1.5 font-mono text-xs leading-relaxed text-foreground/85">
      <span className="text-muted-foreground" aria-hidden>
        $
      </span>
      <span className="line-clamp-3 min-w-0 flex-1 break-all whitespace-pre-wrap">{object}</span>
      <StepStatusMark live={live} failed={false} />
    </span>
  );
  return (
    <div className="text-sm text-foreground/80">
      <p className="flex items-center gap-1.5">
        {verb} a command
        {failed && <span className="text-muted-foreground">·</span>}
        <StepStatusMark live={false} failed={failed} />
      </p>
      {!entry.detail && well}
      {entry.detail && (
        <StepDetailDisclosure detail={entry.detail} at={entry.at}>
          {well}
        </StepDetailDisclosure>
      )}
    </div>
  );
};
