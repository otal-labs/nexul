import { RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useRevertMemory } from "@/hooks/MemoryHooks";
import type { MemoryVersion } from "@/models/MemoryVersion";

interface MemoryVersionRevertButtonProps {
  memoryId: string;
  version: MemoryVersion;
  /** An icon-only button that a row reveals on hover or focus, for a narrow column. */
  compact?: boolean;
}

export const MemoryVersionRevertButton = ({ memoryId, version, compact = false }: MemoryVersionRevertButtonProps) => {
  const revert = useRevertMemory();
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
    <>
      {compact && (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="-my-0.5 size-6 text-muted-foreground opacity-0 hover:text-foreground focus-visible:opacity-100 group-focus-within:opacity-100 group-hover:opacity-100 [@media(hover:none)]:opacity-100"
          aria-label={`Revert to v${version.version}`}
          title={`Revert to v${version.version}`}
          onClick={onRevert}
          loading={revert.isPending}
        >
          <RotateCcw className="size-3.5" aria-hidden />
        </Button>
      )}
      {!compact && (
        <Button type="button" variant="outline" size="sm" onClick={onRevert} loading={revert.isPending}>
          <RotateCcw className="size-4" />
          Revert
        </Button>
      )}
    </>
  );
};
