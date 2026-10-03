import { useState } from "react";

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
      <div className="flex h-9 items-center border-b border-border pr-2 pl-3">
        <FolderToggle name={group.label} open={open} onToggle={() => setCollapsed((c) => !c)} meta={group.memories.length} />
      </div>
      {open && group.memories.length > 0 && (
        <ul className="divide-y divide-border border-b border-border">
          {group.memories.map((memory) => (
            <MemoryListRow key={memory.id} memory={memory} projectToken={projectToken} selected={memory.id === selectedId} />
          ))}
        </ul>
      )}
      {open && group.memories.length === 0 && (
        <EmptyRow className="rounded-none border-0 border-b py-2 pl-13 text-left text-xs">No memories</EmptyRow>
      )}
    </section>
  );
};
