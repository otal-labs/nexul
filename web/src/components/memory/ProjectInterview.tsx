import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewEmptyState } from "@/components/memory/InterviewEmptyState";
import { InterviewRunButton } from "@/components/memory/InterviewRunButton";
import { InterviewThreadSection } from "@/components/memory/InterviewThreadSection";
import { MemoryDetail } from "@/components/memory/MemoryDetail";
import { PageHeader } from "@/components/PageHeader";
import { TrailSection } from "@/components/play/TrailSection";
import { useDeleteMemory, useFetchMemoriesByProject, useUpdateMemory } from "@/hooks/MemoryHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { isInterviewMemory } from "@/models/Memory";
import type { Project } from "@/models/Project";

interface ProjectInterviewProps {
  project: Project;
}

export const ProjectInterview = ({ project }: ProjectInterviewProps) => {
  const canWrite = useHasPermission("memories:write");
  const canDelete = useHasPermission("memories:delete");
  const canClone = useHasPermission("memories:clone");
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
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
        actions={memories && <InterviewRunButton projectId={project.id} hasInterview={interview !== undefined} />}
      />
      <InterviewThreadSection projectId={project.id} />
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
      <TrailSection workspaceId={workspaceId} targetType="interview" targetId={project.id} />
    </div>
  );
};
