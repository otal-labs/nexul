import { ListPaneGroup } from "@/components/listpane/ListPaneGroup";
import { MemoryListRow } from "@/components/memory/MemoryListRow";
import type { MemoryGroup } from "@/components/memory/memoryGroups";

interface MemoryGroupSectionProps {
  group: MemoryGroup;
  projectToken: string;
  selectedId: string | undefined;
}

export const MemoryGroupSection = ({ group, projectToken, selectedId }: MemoryGroupSectionProps) => (
  <ListPaneGroup label={group.label}>
    {group.memories.map((memory) => (
      <MemoryListRow key={memory.id} memory={memory} projectToken={projectToken} selected={memory.id === selectedId} />
    ))}
  </ListPaneGroup>
);
