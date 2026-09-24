import { ClipboardListIcon } from "lucide-react";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { useCreateInterview } from "@/hooks/MemoryHooks";

interface InterviewEmptyStateProps {
  projectId: string;
  canWrite: boolean;
}

export const InterviewEmptyState = ({ projectId, canWrite }: InterviewEmptyStateProps) => {
  const createInterview = useCreateInterview();
  return (
    <EmptyState
      icon={ClipboardListIcon}
      title="No interview yet"
      message="The interview holds this project's rules for agents and goes with every agent turn here. It starts from the workspace's Interview template."
      action={
        canWrite && (
          <Button size="sm" disabled={createInterview.isPending} onClick={() => createInterview.mutate(projectId)}>
            Start from the template
          </Button>
        )
      }
    />
  );
};
