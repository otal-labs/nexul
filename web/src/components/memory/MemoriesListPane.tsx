import { useState } from "react";
import { BrainIcon } from "lucide-react";

import { ListPaneEmpty, ListPaneNoMatch } from "@/components/listpane/ListPaneEmpty";
import { ListPaneHeader } from "@/components/listpane/ListPaneHeader";
import { MemoryGroupSection } from "@/components/memory/MemoryGroupSection";
import { groupMemoriesByFolder } from "@/components/memory/memoryGroups";
import { useCreateMemoryDialog } from "@/hooks/useCreateMemoryDialog";
import type { Memory } from "@/models/Memory";
import { projectToken, type Project } from "@/models/Project";
import { bodySnippet } from "@/utils/BodySnippet";

interface MemoriesListPaneProps {
  memories: Memory[];
  project: Project;
  selectedId: string | undefined;
}

const matches = (memory: Memory, query: string) =>
  [memory.title, memory.when_to_use, bodySnippet(memory.body)].some((text) => text.toLowerCase().includes(query));

export const MemoriesListPane = ({ memories, project, selectedId }: MemoriesListPaneProps) => {
  const [search, setSearch] = useState("");
  const openCreateMemory = useCreateMemoryDialog(project.id);
  const query = search.trim().toLowerCase();
  const searching = query !== "";
  const folders = groupMemoriesByFolder(searching ? memories.filter((m) => matches(m, query)) : memories);
  const groups = searching ? folders.filter((f) => f.memories.length > 0) : folders;

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListPaneHeader
        title="Memories"
        count={memories.length}
        newLabel="New memory"
        onNew={openCreateMemory}
        search={search}
        onSearch={setSearch}
      />
      <div className="min-h-0 flex-1 overflow-y-auto pb-2">
        {memories.length === 0 && <ListPaneEmpty icon={BrainIcon} message="No memories yet" />}
        {memories.length > 0 && groups.length === 0 && <ListPaneNoMatch onClear={() => setSearch("")} />}
        {memories.length > 0 &&
          groups.map((group) => (
            <MemoryGroupSection
              key={group.id}
              group={group}
              projectToken={projectToken(project)}
              selectedId={selectedId}
              forceOpen={searching}
            />
          ))}
      </div>
    </div>
  );
};
