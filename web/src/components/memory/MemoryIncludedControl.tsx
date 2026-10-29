import { LockIcon } from "lucide-react";

import { ActionSwitch } from "@/components/action/ActionSwitch";
import { useSetMemoryFlag } from "@/hooks/MemoryHooks";
import { isDecisionsLogMemory, isInterviewMemory, type Memory } from "@/models/Memory";

interface MemoryIncludedControlProps {
  memory: Memory;
}

// The interview is always sent and the decisions log never is, so theirs is a fixed fact, not a switch.
export const MemoryIncludedControl = ({ memory }: MemoryIncludedControlProps) => {
  const setAlwaysIncluded = useSetMemoryFlag("always_included");
  const interview = isInterviewMemory(memory);
  const decisionsLog = isDecisionsLogMemory(memory);

  return (
    <>
      {(interview || decisionsLog) && (
        <span className="inline-flex items-center gap-1.5 text-foreground">
          <LockIcon className="size-3.5 text-muted-foreground" aria-hidden />
          {interview ? "Always included" : "Not sent every turn"}
        </span>
      )}
      {!interview && !decisionsLog && (
        <label className="inline-flex items-center gap-1.5 text-foreground" title="Sent in every Agent turn and play run">
          <ActionSwitch
            permission="memories:write"
            projectId={memory.project_id}
            verb="change memories"
            size="sm"
            checked={memory.always_included}
            onCheckedChange={(checked) => setAlwaysIncluded.mutate({ memory, value: checked })}
            aria-label="Always included"
          />
          Always included
        </label>
      )}
    </>
  );
};
