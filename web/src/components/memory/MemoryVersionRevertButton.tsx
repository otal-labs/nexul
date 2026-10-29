import { RotateCcw } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFetchMemory, useRevertMemory } from "@/hooks/MemoryHooks";
import type { MemoryVersion } from "@/models/MemoryVersion";

interface MemoryVersionRevertButtonProps {
  memoryId: string;
  version: MemoryVersion;
}

// Icon only, revealed by its row on hover or focus.
export const MemoryVersionRevertButton = ({ memoryId, version }: MemoryVersionRevertButtonProps) => {
  const revert = useRevertMemory();
  const projectId = useFetchMemory(memoryId).data?.project_id;
  const { open: confirm } = useConfirmationDialog();

  const onRevert = async () => {
    const ok = await confirm({
      title: `Revert to version ${version.version}?`,
      message: "This version's content is saved as a new version. Nothing in the history is deleted.",
      confirmLabel: "Revert",
    });
    if (ok) revert.mutate({ id: memoryId, version: version.version });
  };

  return (
    <ActionButton
      permission="memories:write"
      projectId={projectId}
      verb="revert memories"
      icon={RotateCcw}
      type="button"
      variant="ghost"
      size="icon"
      className="-my-0.5 size-5 text-muted-foreground opacity-0 hover:text-foreground focus-visible:opacity-100 group-focus-within:opacity-100 group-hover:opacity-100 [@media(hover:none)]:opacity-100"
      aria-label={`Revert to v${version.version}`}
      title={`Revert to v${version.version}`}
      onClick={onRevert}
      loading={revert.isPending}
    />
  );
};
