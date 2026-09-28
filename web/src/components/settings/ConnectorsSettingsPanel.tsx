import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { ConnectorAppConfigSection } from "@/components/settings/ConnectorAppConfigSection";
import { ConnectorsSection } from "@/components/settings/ConnectorsSection";

interface ConnectorsSettingsPanelProps {
  isInstanceAdmin: boolean;
}

// Connectors comes first so the OAuth callback (?connector=&connected=&error=, no ?tab=) lands where ConnectorsSection reads it.
export const ConnectorsSettingsPanel = ({ isInstanceAdmin }: ConnectorsSettingsPanelProps) => (
  <PageTabs
    label="Connector settings"
    tabs={[
      { value: "connectors", label: "Connectors" },
      { value: "github-app", label: "GitHub App", hidden: !isInstanceAdmin },
    ]}
  >
    <PageTabsContent value="connectors">
      <ConnectorsSection />
    </PageTabsContent>
    <PageTabsContent value="github-app">
      <ConnectorAppConfigSection />
    </PageTabsContent>
  </PageTabs>
);
