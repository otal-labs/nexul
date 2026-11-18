import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RoleLevelsGroup } from "@/components/role/RoleLevelsGroup";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import type { Role } from "@/models/Role";
import { ROLE_SECTIONS, sectionDomains } from "@/models/RoleSection";

interface RoleLevelsSectionProps {
  role: Role;
}

// What the role allows, in the editor's sections and order; the Owner holds the whole catalog.
export const RoleLevelsSection = ({ role }: RoleLevelsSectionProps) => {
  const { data: catalog, error, isPending } = useFetchPermissionCatalog();
  const permissions = role.is_owner_role ? (catalog ?? []).map((entry) => entry.value) : role.permissions;
  const sections = ROLE_SECTIONS.map((section) => ({ ...section, domains: sectionDomains(catalog ?? [], section.area) }));
  return (
    <section aria-labelledby="role-allows-title" className="@container space-y-6 pt-4">
      <h3 id="role-allows-title" className="text-sm font-semibold">
        What {role.name} can do
      </h3>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {catalog &&
        sections.map((section) => (
          <RoleLevelsGroup key={section.area} area={section.area} title={section.title} domains={section.domains} permissions={permissions} />
        ))}
    </section>
  );
};
