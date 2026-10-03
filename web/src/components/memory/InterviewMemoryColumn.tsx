import { NotebookPen } from "lucide-react";

import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewMemoryView } from "@/components/memory/InterviewMemoryView";
import { InterviewRunButton } from "@/components/memory/InterviewRunButton";
import { InterviewRunLine } from "@/components/memory/InterviewRunLine";
import { TrailSection } from "@/components/play/TrailSection";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useActiveTrail } from "@/hooks/TrailHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { isInterviewMemory } from "@/models/Memory";
import { projectToken, type Project } from "@/models/Project";

interface InterviewMemoryColumnProps {
  project: Project;
}

// The memory beside the questions, headed by the latest run's state and, once there is a memory, Regenerate.
export const InterviewMemoryColumn = ({ project }: InterviewMemoryColumnProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: memories, error, isPending } = useFetchMemoriesByProject(project.id);
  const active = useActiveTrail("interview", project.id);
  const memory = memories?.find((m) => isInterviewMemory(m) && m.project_id === project.id);

  return (
    <section aria-label="Interview memory" className="min-w-0 space-y-4">
      <header className="flex min-h-9 flex-wrap items-center gap-3 border-b border-border pb-3">
        <InterviewRunLine projectId={project.id} memory={memory} />
        {(memory || active) && <InterviewRunButton projectId={project.id} label="Regenerate" />}
      </header>
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
