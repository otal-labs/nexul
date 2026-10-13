import { FolderKanban } from "lucide-react";

import { EmptyState } from "@/components/EmptyState";
import { ContinueButton, WelcomeFooter } from "@/components/welcome/WelcomeFooter";
import { WorkspaceProjectsSection } from "@/components/welcome/WorkspaceProjectsSection";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";

const Projects = () => {
  const { data: workspaces } = useFetchWorkspaces();
  return (
    <div className="divide-y divide-border border-y border-border">
      {workspaces?.map((workspace) => (
        <WorkspaceProjectsSection key={workspace.id} workspace={workspace} named={workspaces.length > 1} />
      ))}
    </div>
  );
};

// A person can't add themselves to a project, so with none Skip is the only way on, and the step ticks itself once added.
export const ProjectsStep = () => {
  const inAProject = useWelcomeFlow().marks.projects === "done";
  return (
    <>
      {!inAProject && (
        <EmptyState
          size="compact"
          icon={FolderKanban}
          title="Your team hasn't added you to a project yet"
          message="Ask whoever invited you. This step ticks itself the moment they do."
        />
      )}
      {inAProject && <Projects />}
      <WelcomeFooter skip={!inAProject}>{inAProject && <ContinueButton />}</WelcomeFooter>
    </>
  );
};
