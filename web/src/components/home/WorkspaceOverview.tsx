import { Container } from "@/components/Container";
import { HomeInboxSection } from "@/components/home/HomeInboxSection";
import { HomeRunsSection } from "@/components/home/HomeRunsSection";
import { HomeTicketsSection } from "@/components/home/HomeTicketsSection";
import { WorkspaceOverviewHeader } from "@/components/home/WorkspaceOverviewHeader";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";

// A workspace opens on the person's own work in the project the sidebar shows: what waits on them, what is unread, what runs.
export const WorkspaceOverview = () => {
  const { current: project } = useSidebarProject();
  const can = useAreaAccess(project?.id);
  const canReadPlays = useHasPermission("plays:read");
  return (
    <Container className="@container space-y-8 py-8">
      <WorkspaceOverviewHeader project={project} />
      <div className="grid items-start gap-x-12 gap-y-10 @4xl:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        {project && can?.("tickets") && <HomeTicketsSection project={project} />}
        <div className="grid gap-y-10">
          <HomeInboxSection />
          {project && canReadPlays && <HomeRunsSection project={project} />}
        </div>
      </div>
    </Container>
  );
};
