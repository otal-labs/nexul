import { NotebookPen } from "lucide-react";

import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewMemoryView } from "@/components/memory/InterviewMemoryView";
import { TrailSection } from "@/components/play/TrailSection";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { isWrittenInterview } from "@/models/Memory";
import { projectToken, type Project } from "@/models/Project";

interface InterviewMemorySectionProps {
  project: Project;
}

export const InterviewMemorySection = ({ project }: InterviewMemorySectionProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: memories, error, isPending } = useFetchMemoriesByProject(project.id);
  const memory = memories?.find((m) => isWrittenInterview(m, project.id));

  return (
    <section aria-label="Interview memory" className="min-w-0 space-y-4">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {memories && !memory && (
        <EmptyState
          size="compact"
          icon={NotebookPen}
          title="No memory yet"
          message="The agent writes the memory here once its follow-ups are answered."
        />
      )}
      {memory && <InterviewMemoryView memory={memory} projectToken={projectToken(project)} />}
      <TrailSection workspaceId={workspaceId} targetType="interview" targetId={project.id} rowLayout="stacked" />
    </section>
  );
};
