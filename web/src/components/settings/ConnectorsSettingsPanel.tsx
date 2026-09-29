import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { ConnectorAppConfigSection } from "@/components/settings/ConnectorAppConfigSection";
import { GitHubInstallationsSection } from "@/components/settings/GitHubInstallationsSection";
import { ConnectorsSection } from "@/components/settings/ConnectorsSection";

// Connectors comes first so the OAuth callback (?connector=&connected=&error=, no ?tab=) lands where ConnectorsSection reads it.
export const ConnectorsSettingsPanel = () => (
  <PageTabs
    label="Connector settings"
    tabs={[
      { value: "connectors", label: "Connectors" },
      { value: "github-app", label: "GitHub App" },
    ]}
  >
    <PageTabsContent value="connectors">
      <ConnectorsSection />
    </PageTabsContent>
    <PageTabsContent value="github-app">
      <ConnectorAppConfigSection />
      <GitHubInstallationsSection />
    </PageTabsContent>
  </PageTabs>
);
