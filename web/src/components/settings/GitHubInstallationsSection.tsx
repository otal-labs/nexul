import { Plus } from "lucide-react";
import { Link } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { GitHubInstallationsList } from "@/components/settings/GitHubInstallationsList";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useFetchConnectorAppConfig, useFetchConnectorStatus } from "@/hooks/ConnectorsHooks";
import { githubAppInstallURL } from "@/models/Connectors";

export const GitHubInstallationsSection = () => {
  const { data: app } = useFetchConnectorAppConfig("github");
  const { data: connector, error, isPending } = useFetchConnectorStatus("github");
  if (!app?.configured) return null;

  return (
    <SettingsCard
      id="github-installations"
      title="Installations"
      description="Where the App is installed. Nexul reads repositories in these accounts and organisations."
      footer={
        app.app_slug && (
          <Button asChild variant="outline" size="sm">
            <a href={githubAppInstallURL(app)} target="_blank" rel="noreferrer">
              <Plus className="size-3.5" aria-hidden />
              Add account or organisation
            </a>
          </Button>
        )
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {connector && !connector.status.configured && (
        <p className="text-sm text-muted-foreground">
          Connect GitHub on the{" "}
          <Link to="?tab=connectors" className="underline underline-offset-2 hover:text-foreground">
            Connectors tab
          </Link>{" "}
          to see where the App is installed.
        </p>
      )}
      {connector?.status.configured && <GitHubInstallationsList />}
    </SettingsCard>
  );
};
