import { ActionButton } from "@/components/action/ActionButton";
import { useCancelDeploy } from "@/hooks/DeployHooks";
import { useFetchStack } from "@/hooks/StackHooks";
import { useStackActions } from "@/hooks/useStackActions";
import type { Deploy } from "@/models/Stack";

interface DeployCancelButtonProps {
  deploy: Pick<Deploy, "id" | "stack_id">;
}

export const DeployCancelButton = ({ deploy }: DeployCancelButtonProps) => {
  const cancel = useCancelDeploy();
  const { data: stack } = useFetchStack(deploy.stack_id);
  const answer = useStackActions(stack);
  if (!stack) return null;
  return (
    <ActionButton
      permission="deploys:write"
      projectId={stack.project_id || undefined}
      answer={answer("deploys:write")}
      verb="cancel deploys"
      type="button"
      variant="outline"
      loading={cancel.isPending}
      onClick={() => cancel.mutate(deploy.id)}
    >
      Cancel deploy
    </ActionButton>
  );
};
