import { useMemo } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewChecklistFeed } from "@/components/memory/InterviewChecklistFeed";
import { useFetchInterviewDrafts, useFetchInterviewSources } from "@/hooks/InterviewSourceHooks";
import { useFetchInterviewAnswers, useFetchProjectInterviewTemplate, useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { buildSections } from "@/models/InterviewAnswer";
import { hasInterview } from "@/models/Memory";

interface InterviewChecklistProps {
  projectId: string;
}

export const InterviewChecklist = ({ projectId }: InterviewChecklistProps) => {
  const canWrite = useHasPermission("memories:write");
  const template = useFetchProjectInterviewTemplate(projectId);
  const answers = useFetchInterviewAnswers(projectId);
  const { data: drafts } = useFetchInterviewDrafts(projectId);
  const { data: sources } = useFetchInterviewSources(projectId);
  const { data: memories } = useFetchMemoriesByProject(projectId);
  const hasMemory = !!memories && hasInterview(memories, projectId);
  const isPending = template.isPending || answers.isPending;
  const error = template.error ?? answers.error;
  const sections = useMemo(
    () => template.data && answers.data && buildSections(template.data.questions, answers.data, drafts, sources),
    [template.data, answers.data, drafts, sources],
  );

  return (
    <div className="min-w-0">
      {isPending && !error && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {sections && (
        <InterviewChecklistFeed
          projectId={projectId}
          sections={sections}
          readOnly={!canWrite}
          hasMemory={hasMemory}
          memoryWithoutAnswers={answers.data?.length === 0 && hasMemory}
        />
      )}
    </div>
  );
};
