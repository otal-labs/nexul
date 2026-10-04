import { NotebookPen } from "lucide-react";

import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewAuditButton } from "@/components/memory/InterviewAuditButton";
import { InterviewAuditLine } from "@/components/memory/InterviewAuditLine";
import { InterviewMemoryView } from "@/components/memory/InterviewMemoryView";
import { InterviewRunButton } from "@/components/memory/InterviewRunButton";
import { InterviewRunLine } from "@/components/memory/InterviewRunLine";
import { TrailSection } from "@/components/play/TrailSection";
import { useFetchInterviewAnswers, useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useInterviewTrails } from "@/hooks/InterviewSourceHooks";
import { useActiveTrail } from "@/hooks/TrailHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { answersChangedSinceRun } from "@/models/InterviewAnswer";
import { isWrittenInterview } from "@/models/Memory";
import { projectToken, type Project } from "@/models/Project";

interface InterviewMemoryColumnProps {
  project: Project;
}

// The memory, headed by the latest run's state, Regenerate, and Audit via AI; during a run only that run's own button shows.
export const InterviewMemoryColumn = ({ project }: InterviewMemoryColumnProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: memories, error, isPending } = useFetchMemoriesByProject(project.id);
  const { data: answers } = useFetchInterviewAnswers(project.id);
  const { followUp: trails, audit: audits } = useInterviewTrails(project.id);
  const active = useActiveTrail("interview", project.id);
  const followUpActive = !!active && !!trails?.some((t) => t.id === active.id);
  const auditActive = !!active && !!audits?.some((t) => t.id === active.id);
  const memory = memories?.find((m) => isWrittenInterview(m, project.id));
  const changed = memory && answers && trails ? answersChangedSinceRun(answers, trails) : 0;
  const stale = changed > 0 && !active;

  return (
    <section aria-label="Interview memory" className="min-w-0 space-y-4">
      <header className="flex min-h-9 flex-wrap items-center gap-3 border-b border-border pb-3">
        <InterviewRunLine projectId={project.id} memory={memory} changed={changed} />
        {((memory && !active) || followUpActive) && (
          <InterviewRunButton
            projectId={project.id}
            label={stale ? "Regenerate the memory" : "Regenerate"}
            variant={stale ? "default" : "outline"}
          />
        )}
        {((memory && !active) || auditActive) && <InterviewAuditButton projectId={project.id} />}
        <InterviewAuditLine project={project} />
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
