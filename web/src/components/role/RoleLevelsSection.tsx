import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RoleLevelsGroup } from "@/components/role/RoleLevelsGroup";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import type { Role } from "@/models/Role";
import { ROLE_SECTIONS, sectionDomains } from "@/models/RoleSection";

interface RoleLevelsSectionProps {
  role: Role;
}

// What the role allows, in the editor's sections; the Owner holds the whole catalog.
export const RoleLevelsSection = ({ role }: RoleLevelsSectionProps) => {
  const { data: catalog, error, isPending } = useFetchPermissionCatalog();
  const permissions = role.is_owner_role ? (catalog ?? []).map((entry) => entry.value) : role.permissions;
  return (
    <div className="@container space-y-6">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {catalog &&
        ROLE_SECTIONS.map((section) => (
          <RoleLevelsGroup key={section.area} title={section.title} domains={sectionDomains(catalog, section.area)} permissions={permissions} />
        ))}
    </div>
  );
};
