import { HeaderFacts } from "@/components/HeaderFacts";
import { PageHeader } from "@/components/PageHeader";
import { RolesAndPeopleSection } from "@/components/role/RolesAndPeopleSection";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { countLabel } from "@/models/AccessGrid";

// The Access page for someone who edits roles or gives them but manages no project: the server refuses them the grids, so only Roles and people shows.
export const AccessRolesView = () => {
  const crumb = useWorkspaceCrumb();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: roles } = useFetchWorkspaceRoles(workspaceId);

  return (
    <>
      <PageHeader
        className="mb-6 shrink-0"
        crumbs={[crumb]}
        title="Access"
        meta={roles && <HeaderFacts parts={[countLabel(roles.length, "role", "roles")]} />}
      />
      <RolesAndPeopleSection />
    </>
  );
};
