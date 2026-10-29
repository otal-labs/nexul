import { AccessChannelsTab } from "@/components/accessgrid/AccessChannelsTab";
import { AccessProjectsTab } from "@/components/accessgrid/AccessProjectsTab";
import { PageHeader } from "@/components/PageHeader";
import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { RoleSettingsSection } from "@/components/settings/RoleSettingsSection";
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

// The workspace's Access page once set up: the two grids and the roles as tabs, Public channels for the Owner alone.
export const AccessTabsView = ({ grid, owner }: AccessTabsViewProps) => {
  const crumb = useWorkspaceCrumb();
  const canWriteRoles = useHasPermission("roles:write");
  const wsPath = useWorkspacePath();
  const draft = useAccessDraft(grid);
  useLeaveGuard(draft.projectChanges.length + draft.channelChanges.length > 0, wsPath("/access"), useAccessDraftStore.getState().clear);
  const facts = [
    countLabel(grid.projects.length, "project", "projects"),
    owner && countLabel(grid.channels.length, "public channel", "public channels"),
    countLabel(grid.roles.length, "role", "roles"),
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <>
      <PageHeader className="mb-6 shrink-0" crumbs={[crumb]} title="Access" meta={<span className="font-mono text-xs text-muted-foreground">{facts}</span>} />
      <PageTabs
        label="Access"
        className="min-h-0 flex-1"
        tabs={[
          { value: "projects", label: "Projects" },
          { value: "roles", label: "Roles", hidden: !canWriteRoles },
          { value: "channels", label: "Public channels", hidden: !owner },
        ]}
      >
        <PageTabsContent value="projects" className="flex min-h-0 flex-col">
          <AccessProjectsTab grid={grid} />
        </PageTabsContent>
        <PageTabsContent value="roles" className="flex min-h-0 flex-col">
          <RoleSettingsSection />
        </PageTabsContent>
        <PageTabsContent value="channels" className="flex min-h-0 flex-col">
          <AccessChannelsTab grid={grid} />
        </PageTabsContent>
      </PageTabs>
    </>
  );
};
