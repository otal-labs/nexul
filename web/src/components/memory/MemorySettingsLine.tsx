import { formatUpdatedAgo } from "@/components/doc/docTime";
import { MemoryFolderMenu } from "@/components/memory/MemoryFolderMenu";
import { MemoryIncludedControl } from "@/components/memory/MemoryIncludedControl";
import { isDecisionsLogMemory, isInterviewMemory, type Memory } from "@/models/Memory";

interface MemorySettingsLineProps {
  memory: Memory;
}

// A memory's two settings sit under its title, so the record answers when an agent reads it without the list.
export const MemorySettingsLine = ({ memory }: MemorySettingsLineProps) => (
  <div className="space-y-1">
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[13px]">
      <MemoryFolderMenu memory={memory} />
      <MemoryIncludedControl memory={memory} />
      <span className="font-mono text-xs text-muted-foreground tabular-nums">
        updated {formatUpdatedAgo(memory.updated_at)}
      </span>
    </div>
    {isInterviewMemory(memory) && <p className="text-[13px] text-muted-foreground">Sent in every Agent turn in this project; it can&apos;t be switched off.</p>}
    {isDecisionsLogMemory(memory) && <p className="text-[13px] text-muted-foreground">Read from the memory index when an agent needs the why.</p>}
  </div>
);
