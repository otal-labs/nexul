import { BrainIcon, PlusIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { ListDetailPlaceholder } from "@/components/listpane/ListDetailPlaceholder";
import { useCreateMemoryDialog } from "@/hooks/useCreateMemoryDialog";
import type { Memory } from "@/models/Memory";
import type { Project } from "@/models/Project";

interface MemoriesPlaceholderProps {
  memories: Memory[];
  project: Project;
}

// The open pane before a memory is picked: what the project holds, and the way to add one.
export const MemoriesPlaceholder = ({ memories, project }: MemoriesPlaceholderProps) => {
  const openCreateMemory = useCreateMemoryDialog(project.id);
  const count = memories.length;
  return (
    <ListDetailPlaceholder
      icon={BrainIcon}
      title={count === 0 ? "Nothing to open yet" : "Select a memory"}
      summary={count === 0 ? `${project.name} has no memories.` : `${count} ${count === 1 ? "memory" : "memories"} in ${project.name}`}
      searchable={count > 0}
      action={
        <ActionButton permission="memories:write" verb="create memories" projectId={project.id} size="sm" icon={PlusIcon} onClick={openCreateMemory}>
          New memory
        </ActionButton>
      }
    />
  );
};
