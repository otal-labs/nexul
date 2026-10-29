import { actionState, type ActionGate, type ActionState } from "@nexul/client-core/actionState";

import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { useSessionStore } from "@/stores/sessionStore";
import { useSetupPassStore } from "@/stores/setupPassStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const LIVE: ActionState = { state: "live" };

// Hidden until the viewer's permissions arrive, so a control never flashes live and then locks.
// First run has no account to ask: the setup pass reaches only setup's routes, and the server holds it to them.
export const useActionStates = (gates: ActionGate[]): ActionState[] => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const selectedProjectId = useWorkspaceStore((s) => s.selectedProjectId);
  const hasPass = useSetupPassStore((s) => s.token !== null);
  const signedIn = useSessionStore((s) => s.isLoggedIn);
  const firstRun = hasPass && !signedIn;
  const { data } = useFetchMyRole(firstRun ? "" : selectedWorkspaceId);
  const { data: account } = useFetchMe(!firstRun);
  if (firstRun) return gates.map(() => LIVE);
  return gates.map((gate) => actionState({ me: data, instance: account, ...gate, projectId: gate.projectId ?? selectedProjectId }));
};

export const useActionState = (gate: ActionGate): ActionState => useActionStates([gate])[0] ?? { state: "hidden" };
