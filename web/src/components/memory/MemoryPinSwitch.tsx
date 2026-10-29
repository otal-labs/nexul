import { LockIcon } from "lucide-react";

import { ActionSwitch } from "@/components/action/ActionSwitch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useSetMemoryFlag } from "@/hooks/MemoryHooks";
import { isDecisionsLogMemory, isInterviewMemory, type Memory } from "@/models/Memory";

interface MemoryPinSwitchProps {
  memory: Memory;
  label: string;
}

// Saves on toggle. The interview is always included and the decisions log never is, so theirs is a lock in the switch's
// place, its box as wide as the switch so the dates beside it stay in one column.
export const MemoryPinSwitch = ({ memory, label }: MemoryPinSwitchProps) => {
  const setAlwaysIncluded = useSetMemoryFlag("always_included");
  const interview = isInterviewMemory(memory);
  const fixed = interview || isDecisionsLogMemory(memory);
  const state = interview ? "Always included" : "Not sent every turn";

  return (
    <>
      {fixed && (
        <Tooltip>
          <TooltipTrigger asChild>
            <span tabIndex={0} role="img" aria-label={`${memory.title}: ${state}`} className="flex w-6 justify-center rounded-sm text-muted-foreground">
              <LockIcon className="size-3.5" aria-hidden />
            </span>
          </TooltipTrigger>
          <TooltipContent>{state}</TooltipContent>
        </Tooltip>
      )}
      {!fixed && (
        <ActionSwitch
          permission="memories:write"
          projectId={memory.project_id}
          verb="change memories"
          size="sm"
          checked={memory.always_included}
          onCheckedChange={(checked) => setAlwaysIncluded.mutate({ memory, value: checked })}
          aria-label={label}
          title="Sent in every Agent turn and play run"
        />
      )}
    </>
  );
};
