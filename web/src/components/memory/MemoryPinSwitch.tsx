import { Switch } from "@/components/ui/switch";
import { useSetMemoryAlwaysIncluded } from "@/hooks/MemoryHooks";
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
  const setAlwaysIncluded = useSetMemoryAlwaysIncluded();
  const fixed = isInterviewMemory(memory) || isDecisionsLogMemory(memory);

  return (
    <Switch
      size={size}
      checked={memory.always_included}
      disabled={!canWrite || fixed}
      onCheckedChange={(checked) => setAlwaysIncluded.mutate({ memory, alwaysIncluded: checked })}
      aria-label={label}
      title="Always included in every turn"
    />
  );
};
