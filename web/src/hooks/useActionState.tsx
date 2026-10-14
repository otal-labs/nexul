import { actionState, type ActionGate, type ActionState } from "@nexul/client-core/actionState";

import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Hidden until the viewer's permissions arrive, so a control never flashes live and then locks.
export const useActionState = ({ permission, projectId, answer, verb }: ActionGate): ActionState => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const selectedProjectId = useWorkspaceStore((s) => s.selectedProjectId);
  const { data } = useFetchMyRole(selectedWorkspaceId);
  const { data: account } = useFetchMe();
  return actionState({ me: data, instance: account, permission, projectId: projectId ?? selectedProjectId, answer, verb });
};
