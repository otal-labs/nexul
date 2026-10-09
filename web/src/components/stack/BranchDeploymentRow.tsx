import { Button } from "@/components/ui/button";
import { useDeleteStack } from "@/hooks/StackHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Stack } from "@/models/Stack";

interface BranchDeploymentRowProps {
  deployment: Stack;
}

export const BranchDeploymentRow = ({ deployment }: BranchDeploymentRowProps) => {
  const deleteStack = useDeleteStack();
  const { open: confirm } = useConfirmationDialog();

  // A tear-down deletes the branch's own stack, containers and all; the next matching push makes a new one.
  const tearDown = async () => {
    const ok = await confirm({
      title: `Tear down ${deployment.name}?`,
      message: `Its containers stop and its stack is deleted. The next push to ${deployment.branch ?? "the branch"} deploys it again.`,
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
      <Button variant="ghost" size="sm" className="hover:text-destructive" onClick={() => void tearDown()} loading={deleteStack.isPending}>
        Tear down
      </Button>
    </li>
  );
};
