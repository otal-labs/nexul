import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { ConnectorAppConfigSection } from "@/components/settings/ConnectorAppConfigSection";
import { InstanceAccessSection } from "@/components/settings/InstanceAccessSection";
import { GitHubInstallationsSection } from "@/components/settings/GitHubInstallationsSection";
import { ConnectorsSection } from "@/components/settings/ConnectorsSection";
import { useHasInstancePermission, useIsInstanceOwner } from "@/hooks/AccessHooks";
import { useFetchConnectorAppConfig } from "@/hooks/ConnectorsHooks";

// Connectors comes first so the OAuth callback (?connector=&connected=&error=, no tab segment) lands where ConnectorsSection reads it.
export const ConnectorsSettingsPanel = () => {
  // Managing the App takes connectors:write; the Installations card shows to anyone here once an App is registered.
  const canManageApp = useHasInstancePermission("connectors:write");
  const { data: app } = useFetchConnectorAppConfig("github");
  const showGitHubApp = canManageApp || !!app?.configured;
  const owner = useIsInstanceOwner();

  return (
    <PageTabs
      label="Connector settings"
      tabs={[
        { value: "connectors", label: "Connectors" },
        { value: "github-app", label: "GitHub App", hidden: !showGitHubApp },
        { value: "access", label: "Access", hidden: !owner },
      ]}
    >
      <PageTabsContent value="connectors">
        <ConnectorsSection />
      </PageTabsContent>
      <PageTabsContent value="github-app">
        {canManageApp && <ConnectorAppConfigSection />}
        <GitHubInstallationsSection />
      </PageTabsContent>
      <PageTabsContent value="access">
        <InstanceAccessSection section="connectors" />
      </PageTabsContent>
    </PageTabs>
  );
};
