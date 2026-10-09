import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { MemoryVersionsFeed } from "@/components/memory/MemoryVersionsFeed";

interface MemoryRailProps {
  memoryId: string;
  currentVersion: number;
  canRevert: boolean;
}

// Sits where a doc's "On this page" does; the tabs under the card cover narrower containers, and both read the same queries.
export const MemoryRail = ({ memoryId, currentVersion, canRevert }: MemoryRailProps) => (
  <div className="hidden w-56 shrink-0 @4xl:block">
    <div className="sticky top-6 -mx-2 flex max-h-[calc(100vh-3rem)] flex-col gap-8 overflow-y-auto px-2">
      <AttachmentsSection owner={{ memory_id: memoryId }} actionPlacement="end" />
      <MemoryVersionsFeed memoryId={memoryId} currentVersion={currentVersion} canRevert={canRevert} rowLayout="stacked" />
    </div>
  </div>
);
