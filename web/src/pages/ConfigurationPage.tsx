import { Navigate, useLocation, useParams } from "react-router";

import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { ConfigurationHeaderMeta } from "@/components/settings/ConfigurationHeaderMeta";
import { isSettingsSection, SettingsNav } from "@/components/settings/SettingsNav";
import { SettingsPageContent } from "@/components/settings/SettingsPageContent";
import { useConfigurationSections, useHasInstancePermission } from "@/hooks/AccessHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";

// Same section-per-view shape as ProjectSettingsPage: the :section path segment drives the card, SettingsNav lists sections.
export const ConfigurationPage = () => {
  // Same permission the roles REST endpoints require server-side.
  const canManageRoles = useHasPermission("roles:write");
  const canReadPlays = useHasPermission("plays:read");
  const canWritePlays = useHasPermission("plays:write");
  const canDeletePlays = useHasPermission("plays:delete");
  const canManageWorkspace = useHasPermission("workspaces:write");
  const teamIsInstanceWide = useHasInstancePermission("accounts:read");
  const sections = useConfigurationSections() ?? [];
  const workspaceCrumb = useWorkspaceCrumb();

  const { section: rawSection } = useParams();
  const { search, hash } = useLocation();
  // Team with accounts:read is the instance-wide one on Settings; only a workspace manager's scoped Team stays here.
  const teamMoved = rawSection === "team" && teamIsInstanceWide ? `/settings/team${search}${hash}` : undefined;
  // A section the viewer can't open (unknown, or gated away) falls back to the first they can; Danger zone is never gated, so it's the last resort.
  const fallback = sections.find((candidate) => candidate !== "danger") ?? "danger";
  const section = isSettingsSection(rawSection) && sections.includes(rawSection) ? rawSection : fallback;

  return (
    <Container size="page" className="py-8">
      {teamMoved && <Navigate to={teamMoved} replace />}
      <PageHeader
        className="mb-8"
        crumbs={[workspaceCrumb]}
        title="Configuration"
        meta={<ConfigurationHeaderMeta section={section} />}
      />
      <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:gap-8">
        <SettingsNav active={section} sections={sections} />
        <div className="min-w-0 flex-1 space-y-6">
          <SettingsPageContent
            section={section}
            canManageRoles={canManageRoles}
            canReadPlays={canReadPlays}
            canWritePlays={canWritePlays}
            canDeletePlays={canDeletePlays}
            canManageWorkspace={canManageWorkspace}
          />
        </div>
      </div>
    </Container>
  );
};
