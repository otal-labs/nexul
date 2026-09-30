import { Navigate, useLocation, useParams } from "react-router";

import { Container } from "@/components/Container";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PageHeader } from "@/components/PageHeader";
import { InstanceSettingsContent } from "@/components/settings/InstanceSettingsContent";
import { isSettingsSection } from "@/components/settings/SettingsNav";
import { YourSettingsContent } from "@/components/you/YourSettingsContent";
import {
  DEFAULT_YOUR_SETTINGS_SECTION,
  isYourSettingsSection,
  YourSettingsNav,
} from "@/components/you/YourSettingsNav";
import { useHasInstancePermission, useInstanceSettingsSections, useVisibleSettingsSections } from "@/hooks/AccessHooks";
import { movedSettingsTarget } from "@/utils/SettingsRedirects";

export const YourSettingsPage = () => {
  const { section: rawSection } = useParams();
  const { search, hash } = useLocation();
  const instanceSections = useInstanceSettingsSections();
  const visible = useVisibleSettingsSections();
  const teamIsInstanceWide = useHasInstancePermission("accounts:read");
  // A manager without accounts:read has Team in Configuration, scoped to their workspaces.
  const scopedTeam =
    rawSection === "team" && visible?.includes("team") && !teamIsInstanceWide ? `/configuration/team${search}${hash}` : undefined;
  const moved = movedSettingsTarget(rawSection, search, hash) ?? scopedTeam;
  const instanceSection = isSettingsSection(rawSection) && instanceSections?.includes(rawSection) ? rawSection : undefined;
  // An instance link holds its place until permissions answer, so a deep link never flashes Profile first.
  const resolving = !instanceSections && isSettingsSection(rawSection);
  const section = isYourSettingsSection(rawSection) ? rawSection : DEFAULT_YOUR_SETTINGS_SECTION;

  return (
    <>
      {moved && <Navigate to={moved} replace />}
      {!moved && (
        <Container className="mx-auto max-w-5xl py-10">
          <PageHeader
            className="mb-8"
            eyebrow="You"
            title="Settings"
            subtitle="Your profile, how Nexul looks, and where you're signed in."
          />
          <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:gap-8">
            <YourSettingsNav active={instanceSection ?? section} />
            <div className="min-w-0 flex-1 space-y-6">
              {resolving && <LoadingDisplay />}
              {!resolving && instanceSection && <InstanceSettingsContent section={instanceSection} />}
              {!resolving && !instanceSection && <YourSettingsContent section={section} />}
            </div>
          </div>
        </Container>
      )}
    </>
  );
};
