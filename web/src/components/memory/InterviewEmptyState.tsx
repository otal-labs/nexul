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
      message="The interview holds this project's rules for agents and goes with every agent turn here. Run it to answer one question at a time, or start from the workspace's Interview template and write it yourself."
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
