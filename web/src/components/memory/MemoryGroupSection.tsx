import { useState } from "react";

import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { FolderToggle } from "@/components/listpane/FolderToggle";
import { MemoryListRow } from "@/components/memory/MemoryListRow";
import type { MemoryFolder } from "@/components/memory/memoryGroups";

interface MemoryGroupSectionProps {
  group: MemoryFolder;
  projectToken: string;
  selectedId: string | undefined;
  /** A running search shows every matching folder open, whatever was collapsed. */
  forceOpen: boolean;
}

export const MemoryGroupSection = ({ group, projectToken, selectedId, forceOpen }: MemoryGroupSectionProps) => {
  const [collapsed, setCollapsed] = useState(false);
  const open = forceOpen || !collapsed;
  return (
    <section aria-label={group.label}>
      <div className="flex h-9 items-center pr-3 pl-3">
        <FolderToggle name={group.label} hint={group.hint} open={open} onToggle={() => setCollapsed((c) => !c)} meta={group.memories.length} />
      </div>
      {open && group.memories.length > 0 && (
        <EnterList>
          {group.memories.map((memory) => (
            <MemoryListRow key={memory.id} memory={memory} projectToken={projectToken} selected={memory.id === selectedId} />
          ))}
        </EnterList>
      )}
      {open && group.memories.length === 0 && (
        <EmptyRow className="rounded-none border-0 py-2 pl-13 text-left text-xs">No memories</EmptyRow>
      )}
    </section>
  );
};
