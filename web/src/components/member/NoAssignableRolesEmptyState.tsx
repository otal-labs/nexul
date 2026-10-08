import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/EmptyState";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

export const NoAssignableRolesEmptyState = () => {
  const wsPath = useWorkspacePath();
  return (
    <EmptyState
      role="status"
      className="mt-4"
      title="This workspace has no new roles"
      message="Create one before you can invite anyone."
      action={
        <Button asChild>
          <Link to={wsPath("/configuration/roles")}>Create a role</Link>
        </Button>
      }
    />
  );
};
