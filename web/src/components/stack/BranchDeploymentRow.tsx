import { ActionButton } from "@/components/action/ActionButton";
import { useDeleteStack } from "@/hooks/StackHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useEntityActions } from "@/hooks/useEntityActions";
import type { Stack } from "@/models/Stack";

interface BranchDeploymentRowProps {
  deployment: Stack;
}

export const BranchDeploymentRow = ({ deployment }: BranchDeploymentRowProps) => {
  const deleteStack = useDeleteStack();
  const { open: confirm } = useConfirmationDialog();
  const answer = useEntityActions("stack", deployment.derived_from ?? "", !!deployment.derived_from && !!deployment.project_id);

  // A tear-down deletes the branch's own stack, containers and all; the next matching push makes a new one.
  const tearDown = async () => {
    const ok = await confirm({
      title: `Tear down ${deployment.name}?`,
      message: `Its stack is deleted and its containers are removed. The next push to ${deployment.branch ?? "the branch"} deploys it again.`,
      confirmLabel: "Tear down",
    });
    if (ok) deleteStack.mutate(deployment.id);
  };

  return (
    <li className="flex items-center justify-between gap-3 px-3 py-2.5 text-xs">
      <div className="min-w-0 space-y-0.5">
        <p className="truncate font-mono" title={deployment.name}>
          {deployment.name}
        </p>
        <p className="wrap-anywhere font-mono text-muted-foreground">branch {deployment.branch}</p>
      </div>
      <ActionButton
        permission="stacks:delete"
        projectId={deployment.project_id || undefined}
        answer={answer("stacks:delete")}
        verb="tear down branch deployments"
        variant="ghost"
        size="sm"
        liveClassName="hover:text-destructive"
        onClick={() => void tearDown()}
        loading={deleteStack.isPending}
      >
        Tear down
      </ActionButton>
    </li>
  );
};
