import { useParams } from "react-router";

import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { ConfigurationHeaderMeta } from "@/components/settings/ConfigurationHeaderMeta";
import { isSettingsSection, SettingsNav, sectionLabels } from "@/components/settings/SettingsNav";
import { SettingsPageContent } from "@/components/settings/SettingsPageContent";
import { SettingsShell } from "@/components/settings/SettingsShell";
import { useConfigurationSections } from "@/hooks/AccessHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

// Same section-per-view shape as ProjectSettingsPage: the :section path segment drives the card, SettingsNav lists sections.
export const ConfigurationPage = () => {
  const canManageWorkspace = useHasPermission("workspaces:write");
  const sections = useConfigurationSections() ?? [];
  const workspaceCrumb = useWorkspaceCrumb();
  const wsPath = useWorkspacePath();

  const { section: rawSection } = useParams();
  // A section the viewer can't open (unknown, or gated away) falls back to the first they can; Danger zone is never gated, so it's the last resort.
  const fallback = sections.find((candidate) => candidate !== "danger") ?? "danger";
  const section = isSettingsSection(rawSection) && sections.includes(rawSection) ? rawSection : fallback;

  return (
    <Container size="page" className="py-8">
      <PageHeader
        className="mb-8"
        crumbs={[workspaceCrumb, { label: "Configuration", to: wsPath("/configuration") }]}
        title={sectionLabels[section]}
        meta={<ConfigurationHeaderMeta section={section} />}
      />
      <SettingsShell section={section} nav={<SettingsNav active={section} sections={sections} />}>
        <SettingsPageContent section={section} canManageWorkspace={canManageWorkspace} />
      </SettingsShell>
    </Container>
  );
};
