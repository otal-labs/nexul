import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { OAuthProviderSection } from "@/components/settings/OAuthProviderSection";
import type { InstanceSettings } from "@/models/User";

export const SignInProvidersPanel = ({ settings }: { settings: InstanceSettings }) => (
  <PageTabs
    label="Sign-in providers"
    tabs={[
      { value: "discord", label: "Discord" },
      { value: "google", label: "Google" },
    ]}
  >
    <PageTabsContent value="discord">
      <OAuthProviderSection provider="discord" settings={settings} />
    </PageTabsContent>
    <PageTabsContent value="google">
      <OAuthProviderSection provider="google" settings={settings} />
    </PageTabsContent>
  </PageTabs>
);
