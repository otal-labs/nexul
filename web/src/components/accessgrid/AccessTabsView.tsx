import { AccessChannelsTab } from "@/components/accessgrid/AccessChannelsTab";
import { AccessProjectsTab } from "@/components/accessgrid/AccessProjectsTab";
import { PageHeader } from "@/components/PageHeader";
import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { RolesAndPeopleSection } from "@/components/role/RolesAndPeopleSection";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useAccessDraft } from "@/hooks/useAccessDraft";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useLeaveGuard } from "@/hooks/useLeaveGuard";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useAccessDraftStore } from "@/stores/accessDraftStore";
import { countLabel, type AccessGrid } from "@/models/AccessGrid";

interface AccessTabsViewProps {
  grid: AccessGrid;
  owner: boolean;
}

// The workspace's Access page once set up: the two grids and Roles and people as tabs, Public channels for the Owner alone.
export const AccessTabsView = ({ grid, owner }: AccessTabsViewProps) => {
  const crumb = useWorkspaceCrumb();
  const canWriteRoles = useHasPermission("roles:write");
  const canWriteMembers = useHasPermission("members:write");
  const wsPath = useWorkspacePath();
  const draft = useAccessDraft(grid);
  useLeaveGuard(draft.projectChanges.length + draft.channelChanges.length > 0, wsPath("/access"), useAccessDraftStore.getState().clear);
  const facts = [
    // A count of zero says nothing, so it drops out of the line.
    grid.projects.length > 0 && countLabel(grid.projects.length, "project", "projects"),
    owner && grid.channels.length > 0 && countLabel(grid.channels.length, "public channel", "public channels"),
    // The grids list the roles that can be ticked, never the Owner's, so the count adds it back to match the roles list.
    countLabel(grid.roles.length + 1, "role", "roles"),
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <>
      <PageHeader className="mb-6 shrink-0" crumbs={[crumb]} title="Access" meta={<span>{facts}</span>} />
      <PageTabs
        label="Access"
        className="min-h-0 flex-1"
        tabs={[
          { value: "projects", label: "Projects" },
          { value: "roles", label: "Roles and people", hidden: !canWriteRoles && !canWriteMembers },
          { value: "channels", label: "Public channels", hidden: !owner },
        ]}
      >
        <PageTabsContent value="projects" className="flex min-h-0 flex-col">
          <AccessProjectsTab grid={grid} />
        </PageTabsContent>
        <PageTabsContent value="roles" className="flex min-h-0 flex-col">
          <RolesAndPeopleSection />
        </PageTabsContent>
        <PageTabsContent value="channels" className="flex min-h-0 flex-col">
          <AccessChannelsTab grid={grid} />
        </PageTabsContent>
      </PageTabs>
    </>
  );
};
