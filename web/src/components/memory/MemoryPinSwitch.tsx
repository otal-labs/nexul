import { ActionSwitch } from "@/components/action/ActionSwitch";
import { useSetMemoryFlag } from "@/hooks/MemoryHooks";
import { isDecisionsLogMemory, isInterviewMemory, type Memory } from "@/models/Memory";

interface MemoryPinSwitchProps {
  memory: Memory;
  label: string;
  size?: "sm" | "default";
}

// Saves on toggle; the interview memory is always included and the decisions log never is, so theirs is fixed.
export const MemoryPinSwitch = ({ memory, label, size = "default" }: MemoryPinSwitchProps) => {
  const setAlwaysIncluded = useSetMemoryFlag("always_included");
  const fixed = isInterviewMemory(memory) || isDecisionsLogMemory(memory);

  return (
    <ActionSwitch
      permission="memories:write"
      projectId={memory.project_id}
      verb="change memories"
      size={size}
      checked={memory.always_included}
      disabled={fixed}
      onCheckedChange={(checked) => setAlwaysIncluded.mutate({ memory, value: checked })}
      aria-label={label}
      title="Included in every Agent turn and play run"
    />
  );
};
