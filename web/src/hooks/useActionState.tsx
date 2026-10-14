import { actionState, type ActionGate, type ActionState } from "@nexul/client-core/actionState";

import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Hidden until the viewer's permissions arrive, so a control never flashes live and then locks.
export const useActionStates = (gates: ActionGate[]): ActionState[] => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const selectedProjectId = useWorkspaceStore((s) => s.selectedProjectId);
  const { data } = useFetchMyRole(selectedWorkspaceId);
  const { data: account } = useFetchMe();
  return gates.map((gate) => actionState({ me: data, instance: account, ...gate, projectId: gate.projectId ?? selectedProjectId }));
};

export const useActionState = (gate: ActionGate): ActionState => useActionStates([gate])[0] ?? { state: "hidden" };
