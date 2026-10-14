import { RocketIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { ActionButton } from "@/components/action/ActionButton";
import { useDeployStack } from "@/hooks/StackHooks";
import { useStackActions } from "@/hooks/useStackActions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { deployNowInput, deployPath, type Stack } from "@/models/Stack";

interface StackDeployNowButtonProps {
  stack: Stack;
  // The image this stack currently runs, which a run stack without a repository redeploys.
  image: string | undefined;
  // Unsaved edits are not what a deploy would run with.
  disabled?: boolean;
}

export const StackDeployNowButton = ({ stack, image, disabled }: StackDeployNowButtonProps) => {
  const deploy = useDeployStack();
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const answer = useStackActions(stack);
  const input = deployNowInput(stack, image);
  if (!input) return null;

  return (
    <ActionButton
      permission="deploys:write"
      projectId={stack.project_id || undefined}
      answer={answer("deploys:write")}
      verb="deploy"
      type="button"
      variant="outline"
      size="sm"
      icon={RocketIcon}
      loading={deploy.isPending}
      disabled={disabled}
      title={disabled ? "Save your changes first" : undefined}
      onClick={() => deploy.mutate({ stackId: stack.id, ...input }, { onSuccess: (created) => void navigate(wsPath(deployPath(created))) })}
    >
      Deploy now
    </ActionButton>
  );
};
