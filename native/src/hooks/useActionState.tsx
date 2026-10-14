import { actionState, type ActionGate, type ActionState } from "@nexul/client-core/actionState";

import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";

// Hidden until the viewer's permissions arrive, so a control never flashes live and then locks.
export const useActionState = ({ permission, projectId, answer, verb }: ActionGate): ActionState => {
  const { data } = useFetchMyRole();
  const { data: account } = useFetchMe(true);
  const instance = account?.instance_permissions && { instance_permissions: account.instance_permissions, instance_owner: account.instance_owner === true };
  return actionState({ me: data, instance, permission, projectId, answer, verb });
};
