import { Link } from "react-router";

import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchStacks } from "@/hooks/StackHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

interface NoTestTargetRowProps {
  projectId: string;
}

// Never a fallback to production: say so, and point at the place a separate environment is added.
export const NoTestTargetRow = ({ projectId }: NoTestTargetRowProps) => {
  const canOpenStack = useAreaAccess()?.("stacks") ?? false;
  const wsPath = useWorkspacePath();
  const { data: stacks } = useFetchStacks(projectId);
  const stack = stacks?.find((s) => !s.derived_from);
  return (
    <p role="status" className="rounded-lg border border-border px-4 py-3 text-sm text-muted-foreground">
      No test environment yet. Testing never uses production, so add a branch deploy rule with its own network
      {stack && canOpenStack && (
        <>
          {" in "}
          <Link to={wsPath(`/stacks/${stack.id}/branches`)} className="text-foreground underline underline-offset-4">
            {stack.name}
          </Link>
        </>
      )}
      .
    </p>
  );
};
