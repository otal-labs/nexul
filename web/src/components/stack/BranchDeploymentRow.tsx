import { Button } from "@/components/ui/button";
import { useDeleteStack } from "@/hooks/StackHooks";
import type { Stack } from "@/models/Stack";

interface BranchDeploymentRowProps {
  deployment: Stack;
}

export const BranchDeploymentRow = ({ deployment }: BranchDeploymentRowProps) => {
  const deleteStack = useDeleteStack();
  return (
    <li className="flex items-center justify-between gap-3 px-3 py-2.5 text-xs">
      <div className="min-w-0 space-y-0.5">
        <p className="truncate font-mono">{deployment.name}</p>
        <p className="truncate text-muted-foreground">branch {deployment.branch}</p>
      </div>
      <Button variant="outline" size="sm" onClick={() => deleteStack.mutate(deployment.id)} disabled={deleteStack.isPending}>
        Tear down
      </Button>
    </li>
  );
};
