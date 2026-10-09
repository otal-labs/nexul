import { GatewaysSection } from "@/components/dns/GatewaysSection";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ConnectorsSettingsPanel } from "@/components/settings/ConnectorsSettingsPanel";
import { InstanceUrlSection } from "@/components/settings/InstanceUrlSection";
import { InstanceVersionSection } from "@/components/settings/InstanceVersionSection";
import { SettingsCard } from "@/components/settings/SettingsCard";
import type { SettingsSection } from "@/components/settings/SettingsNav";
import { SignInProvidersPanel } from "@/components/settings/SignInProvidersPanel";
import { InstanceTemplatesSection } from "@/components/templates/InstanceTemplatesSection";
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
    </>
  );
};

interface InstanceSettingsContentProps {
  section: SettingsSection;
}

// No permission gate here: the Settings page only selects a section the viewer may open.
export const InstanceSettingsContent = ({ section }: InstanceSettingsContentProps) => (
  <>
    {section === "instance" && <InstancePanel />}
    {section === "sign-in" && <SignInPanel />}
    {section === "connectors" && <ConnectorsSettingsPanel />}
    {section === "dns" && <GatewaysSection />}
    {section === "team" && <TeamSection />}
    {section === "templates" && <InstanceTemplatesSection />}
  </>
);
