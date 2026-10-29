import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { MemoryVersionsFeed } from "@/components/memory/MemoryVersionsFeed";

interface MemorySectionsProps {
  memoryId: string;
  currentVersion: number;
}

// Drawn in the right rail from a 50rem pane and under the body below it; both read the same queries, so there is one request.
export const MemorySections = ({ memoryId, currentVersion }: MemorySectionsProps) => (
  <>
    <AttachmentsSection owner={{ memory_id: memoryId }} actionPlacement="end" />
    <MemoryVersionsFeed memoryId={memoryId} currentVersion={currentVersion} />
  </>
);
