import { useMemo, useState } from "react";

import { formatUpdatedAgo } from "@/components/doc/docTime";
import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { CloneMemoryDialog } from "@/components/memory/CloneMemoryDialog";
import { MemoryPinSwitch } from "@/components/memory/MemoryPinSwitch";
import { memoryFolderOptions } from "@/components/memory/memoryGroups";
import { useSetMemoryFlag } from "@/hooks/MemoryHooks";
import { useConfirmDeleteMemory } from "@/hooks/useConfirmDeleteMemory";
import { useRowGates } from "@/hooks/useRowGates";
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
  const projectId = memory.project_id;
  const { offered, gates, deleteGate } = useRowGates({
    onClone: { permission: "memories:clone", projectId },
    onDelete: { permission: "memories:delete", projectId },
    moveTo: { permission: "memories:write", projectId, verb: "move memories" },
  });
  const setFooter = useSetMemoryFlag("footer");
  const confirmDelete = useConfirmDeleteMemory();
  const wsPath = useWorkspacePath();
  const snippet = useMemo(() => bodySnippet(memory.body), [memory.body]);
  const onClone = offered("onClone") ? () => setCloneOpen(true) : undefined;
  const onDelete = offered("onDelete") ? () => void confirmDelete(memory, selected) : undefined;
  const moveTo =
    offered("moveTo") && memory.kind === ""
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
        meta={<span className="font-mono text-xs text-muted-foreground tabular-nums">{formatUpdatedAgo(memory.updated_at)}</span>}
        menu={{ moveTo, onClone, onDelete, gates, deleteGate }}
        trailing={<MemoryPinSwitch memory={memory} label={`Always include ${memory.title}`} size="sm" />}
      />
      {cloneOpen && <CloneMemoryDialog memoryId={memory.id} open onClose={() => setCloneOpen(false)} />}
    </>
  );
};
