import { Plus } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { GitHubInstallationRow } from "@/components/settings/GitHubInstallationRow";
import { Button } from "@/components/ui/button";
import { useFetchConnectorAppConfig } from "@/hooks/ConnectorsHooks";
import { useFetchInstallations } from "@/hooks/RepositoryHooks";
import { githubAppInstallURL } from "@/models/Connectors";

export const GitHubInstallationsSection = () => {
  const { data: installations, error, isPending } = useFetchInstallations();
  const { data: app } = useFetchConnectorAppConfig("github");

  return (
    <section aria-labelledby="github-installations-title" className="space-y-3 border-t border-border pt-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 id="github-installations-title" className="text-sm font-medium">
            Accounts Nexul can see
          </h3>
          <p className="text-xs text-muted-foreground">Nexul reads repositories wherever its GitHub App is installed.</p>
        </div>
        {app?.app_slug && (
          <Button asChild variant="outline" size="sm">
            <a href={githubAppInstallURL(app)} target="_blank" rel="noreferrer">
              <Plus className="size-3.5" aria-hidden />
              Add account or organisation
            </a>
          </Button>
        )}
      </div>
      {isPending && <LoadingDisplay className="p-4" />}
      {error && <ErrorDisplay error={error} title="Couldn't load the accounts" className="p-4" />}
      {installations && installations.length === 0 && (
        <EmptyRow>The GitHub App isn't installed on any account you can see yet.</EmptyRow>
      )}
      {installations && installations.length > 0 && (
        <ul className="divide-y divide-border overflow-hidden rounded-md border border-border">
          {installations.map((installation) => (
            <GitHubInstallationRow key={installation.id} installation={installation} />
          ))}
        </ul>
      )}
    </section>
  );
};
