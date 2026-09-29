import { useFetchConnectorAppConfig } from "@/hooks/ConnectorsHooks";
import { githubAppInstallURL } from "@/models/Connectors";

export const RepositoryInstallHint = () => {
  const { data: app } = useFetchConnectorAppConfig("github");

  return (
    <div className="space-y-1 text-xs text-muted-foreground">
      <p>
        Don't see a repository? Nexul only sees accounts where its GitHub App is installed.
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
      <p>Repositories you only collaborate on appear once their owner installs it.</p>
    </div>
  );
};
