import { RocketIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useDeployStack } from "@/hooks/StackHooks";
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
  const can = useAreaAccess(stack.project_id || undefined);
  const input = deployNowInput(stack, image);
  if (!input || !can?.("editDeploys")) return null;

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      loading={deploy.isPending}
      disabled={disabled}
      title={disabled ? "Save your changes first" : undefined}
      onClick={() => deploy.mutate({ stackId: stack.id, ...input }, { onSuccess: (created) => void navigate(wsPath(deployPath(created))) })}
    >
      <RocketIcon aria-hidden />
      Deploy now
    </Button>
  );
};
