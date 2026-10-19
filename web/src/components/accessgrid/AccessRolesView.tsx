import { PageHeader } from "@/components/PageHeader";
import { RoleSettingsSection } from "@/components/settings/RoleSettingsSection";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { countLabel } from "@/models/AccessGrid";

// The Access page for someone who edits roles but manages no project: the server refuses them the grids, so only Roles shows.
export const AccessRolesView = () => {
  const crumb = useWorkspaceCrumb();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: roles } = useFetchWorkspaceRoles(workspaceId);

  return (
    <>
      <PageHeader
        className="mb-6"
        crumbs={[crumb]}
        title="Access"
        meta={roles && <span className="font-mono text-xs text-muted-foreground">{countLabel(roles.length, "role", "roles")}</span>}
      />
      <RoleSettingsSection />
    </>
  );
};
