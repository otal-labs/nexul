import { Plus } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ConnectGitHubPrompt } from "@/components/github/ConnectGitHubPrompt";
import { GitHubInstallationsList } from "@/components/settings/GitHubInstallationsList";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useFetchConnectorAppConfig } from "@/hooks/ConnectorsHooks";
import { useFetchGitHubLink } from "@/hooks/GitHubLinkHooks";
import { githubAppInstallURL } from "@/models/Connectors";

// The accounts the viewer's own GitHub can open, read with their sign-in, so nobody else's account ever lists here.
export const GitHubInstallationsSection = () => {
  const { data: app } = useFetchConnectorAppConfig("github");
  const { data: link, error, isPending } = useFetchGitHubLink();
  if (!app?.configured) return null;

  return (
    <SettingsCard
      id="github-installations"
      title="Installations"
      description="The accounts and organisations your GitHub account can open where the App is installed. A workspace uses one once someone who can open one of its repositories attaches it to a project."
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
      {link && link.state !== "connected" && <ConnectGitHubPrompt state={link.state} />}
      {link?.state === "connected" && <GitHubInstallationsList />}
    </SettingsCard>
  );
};
