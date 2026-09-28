import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { InstanceUrlSection } from "@/components/settings/InstanceUrlSection";
import { InstanceVersionSection } from "@/components/settings/InstanceVersionSection";
import { OAuthProviderSection } from "@/components/settings/OAuthProviderSection";
import type { InstanceSettings } from "@/models/User";

interface InstanceSettingsPanelProps {
  settings: InstanceSettings;
  isInstanceAdmin: boolean;
}

export const InstanceSettingsPanel = ({ settings, isInstanceAdmin }: InstanceSettingsPanelProps) => (
  <PageTabs
    label="Instance settings"
    tabs={[
      { value: "general", label: "General" },
      { value: "sign-in", label: "Sign-in providers", hidden: !isInstanceAdmin },
    ]}
  >
    <PageTabsContent value="general">
      {isInstanceAdmin && <InstanceVersionSection />}
      <InstanceUrlSection settings={settings} />
    </PageTabsContent>
    <PageTabsContent value="sign-in">
      <OAuthProviderSection provider="google" settings={settings} />
      <OAuthProviderSection provider="discord" settings={settings} />
    </PageTabsContent>
  </PageTabs>
);
