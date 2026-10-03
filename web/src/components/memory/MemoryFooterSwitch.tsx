import { Switch } from "@/components/ui/switch";
import { useSetMemoryFlag } from "@/hooks/MemoryHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { isDecisionsLogMemory, isInterviewMemory, type Memory } from "@/models/Memory";

interface MemoryFooterSwitchProps {
  memory: Memory;
  label: string;
}

// Saves on toggle; the interview memory and the decisions log are never footers, so theirs is fixed off.
export const MemoryFooterSwitch = ({ memory, label }: MemoryFooterSwitchProps) => {
  const canWrite = useHasPermission("memories:write");
  const setFooter = useSetMemoryFlag("footer");
  const fixed = isInterviewMemory(memory) || isDecisionsLogMemory(memory);

  return (
    <Switch
      checked={memory.footer && !fixed}
      disabled={!canWrite || fixed}
      onCheckedChange={(checked) => setFooter.mutate({ memory, value: checked })}
      aria-label={label}
      title="Read when a play run ends, to conclude it"
    />
  );
};
