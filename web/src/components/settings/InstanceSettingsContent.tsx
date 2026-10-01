import { GatewaysSection } from "@/components/dns/GatewaysSection";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ConnectorsSettingsPanel } from "@/components/settings/ConnectorsSettingsPanel";
import { InstanceSettingsPanel } from "@/components/settings/InstanceSettingsPanel";
import type { SettingsSection } from "@/components/settings/SettingsNav";
import { SignInProvidersPanel } from "@/components/settings/SignInProvidersPanel";
import { InstanceTemplatesSection } from "@/components/templates/InstanceTemplatesSection";
import { TeamSection } from "@/components/team/TeamSection";
import { useFetchSettings } from "@/hooks/AuthHooks";

const InstancePanel = () => {
  const { data: settings, isPending, error } = useFetchSettings();
  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {settings && <InstanceSettingsPanel settings={settings} />}
    </>
  );
};

const SignInPanel = () => {
  const { data: settings, isPending, error } = useFetchSettings();
  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
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
