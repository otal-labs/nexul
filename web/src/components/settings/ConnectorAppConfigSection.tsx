import { ExternalLink } from "lucide-react";

import { ConnectorAppConfigForm } from "@/components/settings/ConnectorAppConfigForm";
import { ConnectorAppEditDialog } from "@/components/settings/ConnectorAppEditDialog";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Fact } from "@/components/Fact";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchConnectorAppConfig } from "@/hooks/ConnectorsHooks";
import { githubAppURL } from "@/models/Connectors";

// GitHub only: bootstrap seeds this app, so the card shows it and edits it. Other connectors register from their card's "Set up app".
export const ConnectorAppConfigSection = () => {
  const { data: app, isPending, error } = useFetchConnectorAppConfig("github");
  const registered = app?.configured ?? false;

  return (
    <SettingsCard
      id="connector-app-config"
      title="GitHub App"
      description="The GitHub App this instance uses to let people connect their own GitHub account. Register
          your own App and paste its credentials here — Nexul never holds or manages the App itself."
      footer={app && registered && <ConnectorAppEditDialog connectorId="github" current={app} />}
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {app && !registered && <ConnectorAppConfigForm connectorId="github" />}
      {app && registered && (
        <dl className="grid gap-4 sm:grid-cols-3">
          <Fact label="App">
            <a
              href={githubAppURL(app)}
              target="_blank"
              rel="noreferrer"
              className="inline-flex min-w-0 items-center gap-1 font-mono text-xs underline-offset-2 hover:underline"
            >
              <span className="truncate">{app.app_slug}</span>
              <ExternalLink className="size-3 shrink-0" aria-hidden />
            </a>
          </Fact>
          <Fact label="Client ID">
            <span className="truncate font-mono text-xs">{app.client_id}</span>
          </Fact>
          <Fact label="Server">
            <span className="truncate font-mono text-xs">{app.base_url || "github.com"}</span>
          </Fact>
        </dl>
      )}
    </SettingsCard>
  );
};
