import { Info } from "lucide-react";

import { useFetchConnectorAppConfig } from "@/hooks/ConnectorsHooks";
import { useFetchInstallURL } from "@/hooks/RepositoryHooks";
import { githubAppInstallURL } from "@/models/Connectors";

// Why a repository may be missing, which depends on whether Nexul reads GitHub as its App or as the connected account.
export const RepositoryInstallHint = () => {
  const { data: app } = useFetchConnectorAppConfig("github");
  const { data: workspaceInstallURL } = useFetchInstallURL();
  const asApp = !!app?.private_key_set;
  const installURL = workspaceInstallURL ?? (app?.app_slug ? githubAppInstallURL(app) : undefined);

  return (
    <div className="space-y-1 text-xs text-muted-foreground">
      <p>
        {asApp && "Don't see a repository? This workspace lists the repositories of the GitHub accounts assigned to it."}
        {!asApp &&
          "Don't see a repository? Nexul reads GitHub as the account connected in Settings → Connectors, so it lists what that account can open wherever its GitHub App is installed."}
        {installURL && (
          <>
            {" "}
            <a href={installURL} target="_blank" rel="noreferrer" className="underline underline-offset-2 hover:text-foreground">
              Install it on another account or organisation
            </a>
          </>
        )}
      </p>
      {asApp && (
        <p>
          Installing from this link adds the account to this workspace. An installation added from Settings is assigned to
          a workspace there by someone who manages connectors.
        </p>
      )}
      {!asApp && (
        <p>
          A repository in someone else's account appears once its owner installs the App there and gives the connected
          account access to it.
        </p>
      )}
      {!asApp && (
        <p className="flex items-start gap-1.5">
          <Info className="mt-px size-3.5 shrink-0" aria-hidden />
          Only the connected account's repositories are visible until the App's private key is added in Settings →
          Connectors → GitHub App.
        </p>
      )}
    </div>
  );
};
