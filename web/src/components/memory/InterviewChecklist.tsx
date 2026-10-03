import { useMemo } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewChecklistFeed } from "@/components/memory/InterviewChecklistFeed";
import { useFetchInterviewAnswers, useFetchInterviewTemplate, useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { buildSections } from "@/models/InterviewAnswer";
import { hasInterview } from "@/models/Memory";

interface InterviewChecklistProps {
  projectId: string;
}

export const InterviewChecklist = ({ projectId }: InterviewChecklistProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const canWrite = useHasPermission("memories:write");
  const template = useFetchInterviewTemplate(workspaceId);
  const answers = useFetchInterviewAnswers(projectId);
  const { data: memories } = useFetchMemoriesByProject(projectId);
  const isPending = template.isPending || answers.isPending;
  const error = template.error ?? answers.error;
  const sections = useMemo(
    () => template.data && answers.data && buildSections(template.data.questions, answers.data),
    [template.data, answers.data],
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
          memoryWithoutAnswers={answers.data?.length === 0 && !!memories && hasInterview(memories, projectId)}
        />
      )}
    </div>
  );
};
