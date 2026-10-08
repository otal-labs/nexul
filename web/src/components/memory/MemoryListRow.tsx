import { useMemo, useState } from "react";

import { formatUpdatedAgo } from "@/components/doc/docTime";
import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { RowActions } from "@/components/listpane/RowActions";
import { CloneMemoryDialog } from "@/components/memory/CloneMemoryDialog";
import { MemoryPinSwitch } from "@/components/memory/MemoryPinSwitch";
import { memoryFolderOptions } from "@/components/memory/memoryGroups";
import { useSetMemoryFlag } from "@/hooks/MemoryHooks";
import { useConfirmDeleteMemory } from "@/hooks/useConfirmDeleteMemory";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Memory } from "@/models/Memory";
import { memoryPath } from "@/models/Project";
import { bodySnippet } from "@/utils/BodySnippet";

interface MemoryListRowProps {
  memory: Memory;
  projectToken: string;
  selected: boolean;
}

export const MemoryListRow = ({ memory, projectToken, selected }: MemoryListRowProps) => {
  const [cloneOpen, setCloneOpen] = useState(false);
  const canClone = useHasPermission("memories:clone");
  const canDelete = useHasPermission("memories:delete");
  const canWrite = useHasPermission("memories:write");
  const setFooter = useSetMemoryFlag("footer");
  const confirmDelete = useConfirmDeleteMemory();
  const wsPath = useWorkspacePath();
  const snippet = useMemo(() => bodySnippet(memory.body), [memory.body]);
  const onClone = canClone ? () => setCloneOpen(true) : undefined;
  const onDelete = canDelete ? () => void confirmDelete(memory, selected) : undefined;
  const moveTo =
    canWrite && memory.kind === ""
      ? {
          label: "Move to",
          options: memoryFolderOptions,
          currentId: memory.footer ? "footer" : "main",
          onMove: (id: string) => setFooter.mutate({ memory, value: id === "footer" }),
        }
      : undefined;

  return (
    <>
      <ListPaneRow
        to={wsPath(memoryPath(projectToken, memory.id))}
        title={memory.title}
        snippet={snippet}
        selected={selected}
        meta={
          <span className="flex flex-col items-end gap-0.5">
            <span className="font-mono text-[11px] text-muted-foreground tabular-nums">
              {formatUpdatedAgo(memory.updated_at)}
            </span>
            {memory.always_included && (
              <span className="rounded-full bg-muted px-1.5 py-px text-[10px] whitespace-nowrap text-muted-foreground">
                required
              </span>
            )}
          </span>
        }
        actions={
          (moveTo || onClone || onDelete) && (
            <RowActions itemLabel={memory.title} moveTo={moveTo} onClone={onClone} onDelete={onDelete} />
          )
        }
        trailing={<MemoryPinSwitch memory={memory} label={`Require ${memory.title}`} size="sm" />}
      />
      {cloneOpen && <CloneMemoryDialog memoryId={memory.id} open onClose={() => setCloneOpen(false)} />}
    </>
  );
};
