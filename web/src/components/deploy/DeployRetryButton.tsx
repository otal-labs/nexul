import { RotateCwIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchStack, useRetryDeploy } from "@/hooks/StackHooks";
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
  const can = useAreaAccess(stack?.project_id || undefined);
  if (!stack || !can?.("editDeploys") || !isRetryable(deploy)) return null;

  return (
    <Button
      type="button"
      variant="outline"
      loading={retry.isPending}
      title={`Start ${retryTarget(deploy)} again with this stack's current settings`}
      onClick={() => retry.mutate(deploy, { onSuccess: (created) => void navigate(wsPath(deployPath(created))) })}
    >
      <RotateCwIcon aria-hidden />
      Retry
    </Button>
  );
};
