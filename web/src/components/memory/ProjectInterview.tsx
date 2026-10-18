import { InterviewChecklist } from "@/components/memory/InterviewChecklist";
import { InterviewMemorySection } from "@/components/memory/InterviewMemorySection";
import { InterviewRunHeader } from "@/components/memory/InterviewRunHeader";
import { InterviewSourcesSection } from "@/components/memory/InterviewSourcesSection";
import { PageHeader } from "@/components/PageHeader";
import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { HarnessReadinessNote } from "@/components/play/HarnessReadinessNote";
import { useInterviewPlays } from "@/hooks/InterviewSourceHooks";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { boardPath, type Project } from "@/models/Project";

interface ProjectInterviewProps {
  project: Project;
}

export const ProjectInterview = ({ project }: ProjectInterviewProps) => {
  const workspaceCrumb = useWorkspaceCrumb();
  const wsPath = useWorkspacePath();
  const { followUp, draft, audit } = useInterviewPlays(project.id);
  return (
  <div className="space-y-6">
    <PageHeader
      crumbs={[workspaceCrumb, { label: project.name, to: wsPath(boardPath(project)) }]}
      title="Interview"
      meta="Answer questions about how this project works. The agent asks about gaps, then writes the rules every agent turn here follows."
    />
    {(followUp || draft || audit) && <HarnessReadinessNote projectId={project.id} className="text-sm" />}
    <InterviewRunHeader project={project} />
    <PageTabs
      label="Interview sections"
      tabs={[
        { value: "questions", label: "Questions" },
        { value: "sources", label: "Sources" },
        { value: "memory", label: "Memory" },
      ]}
    >
      <PageTabsContent value="questions" className="max-w-3xl">
        <InterviewChecklist projectId={project.id} />
      </PageTabsContent>
      <PageTabsContent value="sources" className="max-w-3xl">
        <InterviewSourcesSection projectId={project.id} />
      </PageTabsContent>
      <PageTabsContent value="memory" className="max-w-3xl">
        <InterviewMemorySection project={project} />
      </PageTabsContent>
    </PageTabs>
  </div>
  );
};
