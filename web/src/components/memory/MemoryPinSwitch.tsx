import { Switch } from "@/components/ui/switch";
import { useSetMemoryFlag } from "@/hooks/MemoryHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { isDecisionsLogMemory, isInterviewMemory, type Memory } from "@/models/Memory";

interface MemoryPinSwitchProps {
  memory: Memory;
  label: string;
  size?: "sm" | "default";
}

// Saves on toggle; the interview memory is always included and the decisions log never is, so theirs is fixed.
export const MemoryPinSwitch = ({ memory, label, size = "default" }: MemoryPinSwitchProps) => {
  const canWrite = useHasPermission("memories:write");
  const setAlwaysIncluded = useSetMemoryFlag("always_included");
  const fixed = isInterviewMemory(memory) || isDecisionsLogMemory(memory);

  return (
    <Switch
      size={size}
      checked={memory.always_included}
      disabled={!canWrite || fixed}
      onCheckedChange={(checked) => setAlwaysIncluded.mutate({ memory, value: checked })}
      aria-label={label}
      title="Always included: every Agent turn and play run names it"
    />
  );
};
