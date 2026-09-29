import { useMemo, useState } from "react";

import { formatUpdatedAgo } from "@/components/doc/docTime";
import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { RowActions } from "@/components/listpane/RowActions";
import { CloneMemoryDialog } from "@/components/memory/CloneMemoryDialog";
import { MemoryPinSwitch } from "@/components/memory/MemoryPinSwitch";
import { useConfirmDeleteMemory } from "@/hooks/useConfirmDeleteMemory";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { isWorkspaceMemory, type Memory } from "@/models/Memory";
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
  const confirmDelete = useConfirmDeleteMemory();
  const snippet = useMemo(() => bodySnippet(memory.body), [memory.body]);
  const onClone = canClone ? () => setCloneOpen(true) : undefined;
  const onDelete = canDelete ? () => void confirmDelete(memory, selected) : undefined;

  return (
    <>
      <ListPaneRow
        to={memoryPath(isWorkspaceMemory(memory) ? "" : projectToken, memory.id)}
        title={memory.title}
        snippet={snippet}
        selected={selected}
        meta={
          <>
            {memory.always_included && (
              <span className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] whitespace-nowrap text-muted-foreground">
                always in context
              </span>
            )}
            <span className="font-mono text-[11px] text-muted-foreground tabular-nums">
              {formatUpdatedAgo(memory.updated_at)}
            </span>
          </>
        }
        actions={(onClone || onDelete) && <RowActions itemLabel={memory.title} onClone={onClone} onDelete={onDelete} />}
        trailing={<MemoryPinSwitch memory={memory} label={`Always include ${memory.title}`} size="sm" />}
      />
      {cloneOpen && <CloneMemoryDialog memoryId={memory.id} open onClose={() => setCloneOpen(false)} />}
    </>
  );
};
