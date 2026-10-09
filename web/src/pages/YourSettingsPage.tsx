import { Navigate, useLocation, useParams } from "react-router";

import { Container } from "@/components/Container";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PageHeader } from "@/components/PageHeader";
import { SettingsHeaderMeta } from "@/components/settings/SettingsHeaderMeta";
import { InstanceSettingsContent } from "@/components/settings/InstanceSettingsContent";
import { isSettingsSection } from "@/components/settings/SettingsNav";
import { YourSettingsContent } from "@/components/you/YourSettingsContent";
import { SettingsShell } from "@/components/settings/SettingsShell";
import {
  DEFAULT_YOUR_SETTINGS_SECTION,
  isYourSettingsSection,
  YourSettingsNav,
} from "@/components/you/YourSettingsNav";
import { useHasInstancePermission, useInstanceSettingsSections, useVisibleSettingsSections } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

export const YourSettingsPage = () => {
  const { section: rawSection } = useParams();
  const { search, hash } = useLocation();
  const instanceSections = useInstanceSettingsSections();
  const visible = useVisibleSettingsSections();
  const teamIsInstanceWide = useHasInstancePermission("accounts:read");
  const wsPath = useWorkspacePath();
  // A manager without accounts:read has Team in Configuration, scoped to their workspaces.
  const moved =
    rawSection === "team" && visible?.includes("team") && !teamIsInstanceWide
      ? `${wsPath("/configuration/team")}${search}${hash}`
      : undefined;
  const instanceSection = isSettingsSection(rawSection) && instanceSections?.includes(rawSection) ? rawSection : undefined;
  // An instance link holds its place until permissions answer, so a deep link never flashes Profile first.
  const resolving = !instanceSections && isSettingsSection(rawSection);
  const section = isYourSettingsSection(rawSection) ? rawSection : DEFAULT_YOUR_SETTINGS_SECTION;

  return (
    <>
      {moved && <Navigate to={moved} replace />}
      {!moved && (
        <Container size="page" className="py-8">
          <PageHeader
            className="mb-8"
            title="Settings"
            meta={!resolving && <SettingsHeaderMeta section={instanceSection ?? section} />}
          />
          <SettingsShell section={instanceSection ?? section} nav={<YourSettingsNav active={instanceSection ?? section} />}>
            {resolving && <LoadingDisplay />}
            {!resolving && instanceSection && <InstanceSettingsContent section={instanceSection} />}
            {!resolving && !instanceSection && <YourSettingsContent section={section} />}
          </SettingsShell>
        </Container>
      )}
    </>
  );
};
