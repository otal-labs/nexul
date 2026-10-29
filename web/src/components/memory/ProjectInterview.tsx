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
  <div className="@container/interview max-w-[68rem] space-y-6">
    <PageHeader
      crumbs={[workspaceCrumb, { label: project.name, to: wsPath(boardPath(project)) }]}
      title="Interview"
      meta={
        <span className="max-w-[64ch]">
          Answer questions about how this project works. The agent asks about gaps, then writes the rules every agent turn here follows.
        </span>
      }
    />
    {(followUp || draft || audit) && <HarnessReadinessNote projectId={project.id} className="text-sm" />}
    <InterviewRunHeader project={project} />
    <PageTabs
      label="Interview sections"
      tabs={[
        { value: "questions", label: "Questions" },
        { value: "memory", label: "Memory" },
      ]}
    >
      <PageTabsContent value="questions" className="@container">
        <div className="grid items-start gap-x-12 gap-y-10 @min-[60rem]:grid-cols-[minmax(0,1fr)_20rem]">
          <InterviewChecklist projectId={project.id} />
          <InterviewSourcesSection projectId={project.id} />
        </div>
      </PageTabsContent>
      <PageTabsContent value="memory" className="@container">
        <InterviewMemorySection project={project} />
      </PageTabsContent>
    </PageTabs>
  </div>
  );
};
