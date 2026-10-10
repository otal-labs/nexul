import { useFetchConnectorAppConfig } from "@/hooks/ConnectorsHooks";
import { githubAppInstallURL } from "@/models/Connectors";

export const RepositoryInstallHint = () => {
  const { data: app } = useFetchConnectorAppConfig("github");

  return (
    <div className="space-y-1 text-xs text-muted-foreground">
      <p>
        Don't see a repository? Nexul reads GitHub as the account connected in Settings → Connectors, so it lists
        what that account can open wherever its GitHub App is installed.
        {app?.app_slug && (
          <>
            {" "}
            <a
              href={githubAppInstallURL(app)}
              target="_blank"
              rel="noreferrer"
              className="underline underline-offset-2 hover:text-foreground"
            >
              Install it on another account or organisation
            </a>
          </>
        )}
      </p>
      <p>
        A repository in someone else's account appears once its owner installs the App there and gives the connected
        account access to it.
      </p>
    </div>
  );
};
