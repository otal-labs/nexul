import { InterviewChecklist } from "@/components/memory/InterviewChecklist";
import { InterviewMemoryColumn } from "@/components/memory/InterviewMemoryColumn";
import { PageHeader } from "@/components/PageHeader";
import type { Project } from "@/models/Project";

interface ProjectInterviewProps {
  project: Project;
}

// The columns follow the content area, not the window: with the sidebar open at 1024px the memory drops under.
export const ProjectInterview = ({ project }: ProjectInterviewProps) => (
  <div className="space-y-10">
    <PageHeader
      eyebrow={project.name}
      title="Interview"
      subtitle="Answer a few questions about how this project works. The agent asks about any gaps, then writes the rules every agent turn here follows."
    />
    <div className="@container">
      <div className="grid gap-12 @5xl:grid-cols-[minmax(0,36rem)_minmax(0,1fr)] @5xl:items-start @5xl:gap-10">
        <InterviewChecklist projectId={project.id} />
        <InterviewMemoryColumn project={project} />
      </div>
    </div>
  </div>
);
