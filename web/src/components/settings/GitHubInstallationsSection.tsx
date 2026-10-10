import { Plus } from "lucide-react";
import { Link } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { GitHubInstallationsList } from "@/components/settings/GitHubInstallationsList";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useFetchConnectorAppConfig, useFetchConnectorStatus } from "@/hooks/ConnectorsHooks";
import { useTabPath } from "@/hooks/useTabPath";
import { githubAppInstallURL } from "@/models/Connectors";

export const GitHubInstallationsSection = () => {
  const { data: app } = useFetchConnectorAppConfig("github");
  const { data: connector, error, isPending } = useFetchConnectorStatus("github");
  const { tabPath } = useTabPath();
  if (!app?.configured) return null;

  return (
    <SettingsCard
      id="github-installations"
      title="Installations"
      description={
        app.private_key_set
          ? "Nexul reads these accounts and organisations as the App. Each one's repositories list in the workspaces it is assigned to."
          : "Nexul reads repositories in these accounts and organisations as the connected account, so it sees only what that account can open."
      }
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
      {connector && !connector.status.configured && !app.private_key_set && (
        <p className="text-sm text-muted-foreground">
          Connect GitHub on the{" "}
          <Link to={tabPath()} className="underline underline-offset-2 hover:text-foreground">
            Connectors tab
          </Link>{" "}
          to see where the App is installed.
        </p>
      )}
      {(app.private_key_set || connector?.status.configured) && <GitHubInstallationsList />}
    </SettingsCard>
  );
};
