import { Link } from "react-router";

import { useFetchStacks } from "@/hooks/StackHooks";

interface NoTestTargetRowProps {
  projectId: string;
}

// Never a fallback to production: say so, and point at the place a separate environment is added.
export const NoTestTargetRow = ({ projectId }: NoTestTargetRowProps) => {
  const { data: stacks } = useFetchStacks(projectId);
  const stack = stacks?.find((s) => !s.derived_from);
  return (
    <p role="status" className="rounded-lg border border-border px-4 py-3 text-sm text-muted-foreground">
      No test environment yet. Production is never used for testing, so add a deploy branch on its own network
      {stack && (
        <>
          {" in "}
          <Link to={`/stacks/${stack.id}?section=branches`} className="text-foreground underline underline-offset-4">
            {stack.name}
          </Link>
        </>
      )}
      .
    </p>
  );
};
