import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewChecklistFeed } from "@/components/memory/InterviewChecklistFeed";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useActionState } from "@/hooks/useActionState";
import { useInterviewSections } from "@/hooks/useInterviewSections";
import { hasInterview } from "@/models/Memory";

interface InterviewChecklistProps {
  projectId: string;
}

export const InterviewChecklist = ({ projectId }: InterviewChecklistProps) => {
  const canWrite = useActionState({ permission: "memories:write", projectId }).state === "live";
  const { sections, answers, isPending, error } = useInterviewSections(projectId);
  const { data: memories } = useFetchMemoriesByProject(projectId);
  const hasMemory = !!memories && hasInterview(memories, projectId);

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
          memoryWithoutAnswers={answers?.length === 0 && hasMemory}
        />
      )}
    </div>
  );
};
