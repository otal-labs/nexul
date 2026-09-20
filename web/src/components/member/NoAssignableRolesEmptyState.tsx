import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/EmptyState";

export const NoAssignableRolesEmptyState = () => (
  <EmptyState
    role="status"
    className="mt-4"
    title="This workspace has no new roles"
    message="Create one before you can invite anyone."
    action={
      <Button asChild>
        <Link to="/settings?section=roles">Create a role</Link>
      </Button>
    }
  />
);
