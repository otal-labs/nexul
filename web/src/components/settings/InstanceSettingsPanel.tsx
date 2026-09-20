import { InstanceUrlSection } from "@/components/settings/InstanceUrlSection";
import { InstanceVersionSection } from "@/components/settings/InstanceVersionSection";
import { OAuthProviderSection } from "@/components/settings/OAuthProviderSection";
import type { InstanceSettings } from "@/models/User";

interface InstanceSettingsPanelProps {
  settings: InstanceSettings;
  isInstanceAdmin: boolean;
}

export const InstanceSettingsPanel = ({ settings, isInstanceAdmin }: InstanceSettingsPanelProps) => (
  <>
    {isInstanceAdmin && <InstanceVersionSection />}
    <InstanceUrlSection settings={settings} />
    {isInstanceAdmin && <OAuthProviderSection provider="google" settings={settings} />}
    {isInstanceAdmin && <OAuthProviderSection provider="discord" settings={settings} />}
  </>
);
