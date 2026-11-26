import { useParams } from "react-router";

import { Container } from "@/components/Container";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PageHeader } from "@/components/PageHeader";
import { SettingsHeaderMeta } from "@/components/settings/SettingsHeaderMeta";
import { InstanceSettingsContent } from "@/components/settings/InstanceSettingsContent";
import { isSettingsSection, sectionLabels as instanceLabels } from "@/components/settings/SettingsNav";
import { YourSettingsContent } from "@/components/you/YourSettingsContent";
import { SettingsShell } from "@/components/settings/SettingsShell";
import {
  DEFAULT_YOUR_SETTINGS_SECTION,
  isYourSettingsSection,
  sectionLabels,
  YourSettingsNav,
} from "@/components/you/YourSettingsNav";
import { useHasInstancePermission, useInstanceSettingsSections } from "@/hooks/AccessHooks";
import { MOVE_SIGN_INS_TAB } from "@/models/Access";

export const YourSettingsPage = () => {
  const { section: rawSection, tab } = useParams();
  const instanceSections = useInstanceSettingsSections();
  const readsProviders = useHasInstancePermission("instance:read");
  const instanceSection = isSettingsSection(rawSection) && instanceSections?.includes(rawSection) ? rawSection : undefined;
  // An instance link holds its place until permissions answer, so a deep link never flashes Profile first.
  const resolving = !instanceSections && isSettingsSection(rawSection);
  const section = isYourSettingsSection(rawSection) ? rawSection : DEFAULT_YOUR_SETTINGS_SECTION;
  // The open section names the page, so the title says where you are instead of repeating "Settings".
  const yourTitle = resolving ? "Settings" : sectionLabels[section];
  // A holder of sign_ins:move alone opens Sign-in providers on its Move sign-ins page, the one part they may read.
  const moving = instanceSection === "sign-in" && (tab === MOVE_SIGN_INS_TAB || !readsProviders);
  const instanceTitle = instanceSection && (moving ? "Move sign-ins" : instanceLabels[instanceSection]);
  const title = instanceTitle || yourTitle;
  const crumbs = [{ label: "Settings", to: "/settings" }, ...(moving && readsProviders ? [{ label: instanceLabels["sign-in"], to: "/settings/sign-in" }] : [])];

  return (
    <Container size="page" className="py-8">
      <PageHeader
        className="mb-8"
        crumbs={crumbs}
        title={title}
        meta={!resolving && <SettingsHeaderMeta section={instanceSection ?? section} moving={moving} />}
      />
      <SettingsShell section={instanceSection ?? section} nav={<YourSettingsNav active={instanceSection ?? section} />}>
        {resolving && <LoadingDisplay />}
        {!resolving && instanceSection && <InstanceSettingsContent section={instanceSection} moving={moving} />}
        {!resolving && !instanceSection && <YourSettingsContent section={section} />}
      </SettingsShell>
    </Container>
  );
};
