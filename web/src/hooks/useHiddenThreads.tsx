import { useHiddenThreadStore } from "@/stores/hiddenThreadStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const NONE: string[] = [];

// The thread ids the viewer removed from the sidebar in the selected workspace, kept per browser.
export const useHiddenThreads = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const hiddenIds = useHiddenThreadStore((s) => s.hidden[workspaceId] ?? NONE);
  const toggleHidden = useHiddenThreadStore((s) => s.toggleHidden);
  return { hiddenIds, toggle: (conversationId: string) => toggleHidden(workspaceId, conversationId) };
};
