import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/EmptyState";
import { PageHeader } from "@/components/PageHeader";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

export const BoardNotFoundState = () => {
  const wsPath = useWorkspacePath();
  return (
    <>
      <PageHeader title="Board" subtitle="Every ticket in its lane, traffic optional." />
      <EmptyState
        title="Project not found"
        action={
          <Button asChild size="sm">
            <Link to={wsPath("/board")}>Go to your board</Link>
          </Button>
        }
      />
    </>
  );
};
