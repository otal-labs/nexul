import { groupPlaysByType } from "@/components/play/playGroups";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { useMinWidth } from "@/hooks/useMinWidth";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { agentPath } from "@/models/Agent";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Where the Agent page opens with nothing picked: the list's first play from 1024px, where the open pane sits beside the list; below that the list is the page.
export const useFirstPlayPath = () => {
  const wide = useMinWidth(1024);
  const can = useAreaAccess();
  const wsPath = useWorkspacePath();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const reads = wide && !!can?.("plays");
  const { data: plays, isPending } = useFetchWorkspacePlays(reads ? workspaceId : "");
  const first = plays && groupPlaysByType(plays)[0]?.plays[0];
  return {
    pending: wide && (can === undefined || (reads && isPending)),
    to: first ? wsPath(agentPath("plays", first.id)) : undefined,
  };
};
