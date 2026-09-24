import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewEmptyState } from "@/components/memory/InterviewEmptyState";
import { MemoryDetail } from "@/components/memory/MemoryDetail";
import { PageHeader } from "@/components/PageHeader";
import { useDeleteMemory, useFetchMemoriesByProject, useUpdateMemory } from "@/hooks/MemoryHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { isInterviewMemory } from "@/models/Memory";
import type { Project } from "@/models/Project";

interface ProjectInterviewProps {
  project: Project;
}

export const ProjectInterview = ({ project }: ProjectInterviewProps) => {
  const canWrite = useHasPermission("memories:write");
  const canDelete = useHasPermission("memories:delete");
  const canClone = useHasPermission("memories:clone");
  const { data: memories, error, isPending } = useFetchMemoriesByProject(project.id);
  const updateMemory = useUpdateMemory();
  const deleteMemory = useDeleteMemory();
  const interview = memories?.find((memory) => isInterviewMemory(memory) && memory.project_id === project.id);

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow={project.name}
        title="Interview"
        subtitle="The rules every agent turn in this project follows: stack, paradigm, testing, principles, and vocabulary."
      />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {memories && !interview && <InterviewEmptyState projectId={project.id} canWrite={canWrite} />}
      {interview && (
        <MemoryDetail
          key={`${interview.id}:${interview.version}`}
          memory={interview}
          canWrite={canWrite}
          canDelete={canDelete}
          canClone={canClone}
          saving={updateMemory.isPending}
          onSave={(input) => updateMemory.mutate({ id: interview.id, ...input })}
          onDelete={() => deleteMemory.mutate(interview.id)}
        />
      )}
    </div>
  );
};
