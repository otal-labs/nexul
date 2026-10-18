import { InterviewAuditButton } from "@/components/memory/InterviewAuditButton";
import { InterviewAuditLine } from "@/components/memory/InterviewAuditLine";
import { InterviewRunButton } from "@/components/memory/InterviewRunButton";
import { InterviewRunLine } from "@/components/memory/InterviewRunLine";
import { useFetchInterviewAnswers, useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useInterviewTrails } from "@/hooks/InterviewSourceHooks";
import { useActiveTrail } from "@/hooks/TrailHooks";
import { answersChangedSinceRun } from "@/models/InterviewAnswer";
import { isWrittenInterview } from "@/models/Memory";
import type { Project } from "@/models/Project";

interface InterviewRunHeaderProps {
  project: Project;
}

// The latest run's state, Regenerate, and Audit via AI above every tab; during a run only that run's own button shows.
export const InterviewRunHeader = ({ project }: InterviewRunHeaderProps) => {
  const { data: memories } = useFetchMemoriesByProject(project.id);
  const { data: answers } = useFetchInterviewAnswers(project.id);
  const { followUp: trails, audit: audits } = useInterviewTrails(project.id);
  const active = useActiveTrail("interview", project.id);
  const followUpActive = !!active && !!trails?.some((t) => t.id === active.id);
  const auditActive = !!active && !!audits?.some((t) => t.id === active.id);
  const memory = memories?.find((m) => isWrittenInterview(m, project.id));
  const changed = memory && answers && trails ? answersChangedSinceRun(answers, trails) : 0;
  const stale = changed > 0 && !active;

  return (
    <header className="flex min-h-9 flex-wrap items-center gap-3">
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
  );
};
