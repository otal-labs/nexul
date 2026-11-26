import { GatewaysSection } from "@/components/dns/GatewaysSection";
import { TunnelsSection } from "@/components/dns/TunnelsSection";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ArchivedWorkspacesSection } from "@/components/settings/ArchivedWorkspacesSection";
import { ConnectorsSettingsPanel } from "@/components/settings/ConnectorsSettingsPanel";
import { InstanceAccessSection } from "@/components/settings/InstanceAccessSection";
import { InstanceUrlSection } from "@/components/settings/InstanceUrlSection";
import { InstanceVersionSection } from "@/components/settings/InstanceVersionSection";
import { AuditLogSection } from "@/ee/components/AuditLogSection";
import { BrandingSection } from "@/ee/components/BrandingSection";
import { LicenseSection } from "@/ee/components/LicenseSection";
import { SignInMovesPanel } from "@/ee/components/SignInMovesPanel";
import { SsoSection } from "@/ee/components/SsoSection";
import { SettingsCard } from "@/components/settings/SettingsCard";
import type { SettingsSection } from "@/components/settings/SettingsNav";
import { SignInProvidersPanel } from "@/components/settings/SignInProvidersPanel";
import { InstanceTemplatesSection } from "@/components/templates/InstanceTemplatesSection";
import { TemplateRolesSection } from "@/components/templates/TemplateRolesSection";
import { TeamSection } from "@/components/team/TeamSection";
import { useFetchSettings } from "@/hooks/AuthHooks";

// The version reads its own endpoint, so it shows whether or not the instance settings load.
const InstancePanel = () => {
  const { data: settings, isPending, error } = useFetchSettings();
  return (
    <>
      <InstanceVersionSection />
      {isPending && (
        <SettingsCard id="instance" title="Address">
          <LoadingDisplay />
        </SettingsCard>
      )}
      {error && (
        <SettingsCard id="instance" title="Address">
          <ErrorDisplay error={error} />
        </SettingsCard>
      )}
      {settings && <InstanceUrlSection settings={settings} />}
      <LicenseSection />
      <BrandingSection />
      <AuditLogSection />
    </>
  );
};

const SignInPanel = () => {
  const { data: settings, isPending, error } = useFetchSettings();
  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && (
        <SettingsCard id="sign-in" title="Sign-in providers">
          <ErrorDisplay error={error} />
        </SettingsCard>
      )}
      {settings && <SignInProvidersPanel settings={settings} />}
      <SsoSection />
    </>
  );
};

interface InstanceSettingsContentProps {
  section: SettingsSection;
  // Sign-in providers opens its Move sign-ins page instead of the providers.
  moving?: boolean;
}

// No permission gate here: the Settings page only selects a section the viewer may open. Connectors keeps Access in a tab.
export const InstanceSettingsContent = ({ section, moving = false }: InstanceSettingsContentProps) => (
  <>
    {section === "instance" && <InstancePanel />}
    {section === "instance" && <ArchivedWorkspacesSection />}
    {section === "sign-in" && !moving && <SignInPanel />}
    {section === "sign-in" && moving && <SignInMovesPanel />}
    {section === "connectors" && <ConnectorsSettingsPanel />}
    {section === "dns" && <GatewaysSection />}
    {section === "dns" && <TunnelsSection />}
    {section === "team" && <TeamSection />}
    {section === "templates" && <InstanceTemplatesSection />}
    {section === "templates" && <TemplateRolesSection />}
    {section !== "connectors" && <InstanceAccessSection section={section} />}
  </>
);
