import { usePinnedDocStore } from "@/stores/pinnedDocStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const NONE: string[] = [];

// The viewer's pinned doc ids for the selected workspace, newest pin first, kept per browser.
export const useDocPins = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const pinnedIds = usePinnedDocStore((s) => s.pinned[workspaceId] ?? NONE);
  const togglePin = usePinnedDocStore((s) => s.togglePin);
  return { pinnedIds, toggle: (docId: string) => togglePin(workspaceId, docId) };
};
