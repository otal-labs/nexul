import { RotateCwIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { ActionButton } from "@/components/action/ActionButton";
import { useFetchStack, useRetryDeploy } from "@/hooks/StackHooks";
import { useStackActions } from "@/hooks/useStackActions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { deployCommit, deployPath, isRetryable, type Deploy } from "@/models/Stack";

interface DeployRetryButtonProps {
  deploy: Deploy;
}

const retryTarget = (deploy: Deploy): string => {
  const commit = deployCommit(deploy);
  if (commit) return `commit ${commit.slice(0, 7)}`;
  return deploy.ref || deploy.image;
};

export const DeployRetryButton = ({ deploy }: DeployRetryButtonProps) => {
  const retry = useRetryDeploy();
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { data: stack } = useFetchStack(deploy.stack_id);
  const answer = useStackActions(stack);
  if (!stack || !isRetryable(deploy)) return null;

  return (
    <ActionButton
      permission="deploys:write"
      projectId={stack.project_id || undefined}
      answer={answer("deploys:write")}
      verb="retry deploys"
      type="button"
      variant="outline"
      icon={RotateCwIcon}
      loading={retry.isPending}
      title={`Start ${retryTarget(deploy)} again with this stack's current settings`}
      onClick={() => retry.mutate(deploy, { onSuccess: (created) => void navigate(wsPath(deployPath(created))) })}
    >
      Retry
    </ActionButton>
  );
};
