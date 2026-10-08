import { InterviewChecklist } from "@/components/memory/InterviewChecklist";
import { InterviewMemoryColumn } from "@/components/memory/InterviewMemoryColumn";
import { InterviewSourcesSection } from "@/components/memory/InterviewSourcesSection";
import { PageHeader } from "@/components/PageHeader";
import { HarnessReadinessNote } from "@/components/play/HarnessReadinessNote";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Project } from "@/models/Project";

interface ProjectInterviewProps {
  project: Project;
}

// The columns follow the content area, not the window: with the sidebar open at 1024px the memory drops under.
export const ProjectInterview = ({ project }: ProjectInterviewProps) => {
  const workspaceCrumb = useWorkspaceCrumb();
  const wsPath = useWorkspacePath();
  return (
  <div className="space-y-10">
    <PageHeader
      crumbs={[workspaceCrumb, { label: project.name, to: wsPath(`/board/${project.id}`) }]}
      title="Interview"
      meta="Answer questions about how this project works. The agent asks about gaps, then writes the rules every agent turn here follows."
    />
    <div className="@container space-y-6">
      <HarnessReadinessNote projectId={project.id} className="text-sm" />
      <div className="grid gap-12 @5xl:grid-cols-[minmax(0,36rem)_minmax(0,1fr)] @5xl:items-start @5xl:gap-10">
        <div className="min-w-0 space-y-6">
          <InterviewSourcesSection projectId={project.id} />
          <InterviewChecklist projectId={project.id} />
        </div>
        <InterviewMemoryColumn project={project} />
      </div>
    </div>
  </div>
  );
};
