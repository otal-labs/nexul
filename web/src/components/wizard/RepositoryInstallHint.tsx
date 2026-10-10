import { useFetchConnectorAppConfig } from "@/hooks/ConnectorsHooks";
import { githubAppInstallURL } from "@/models/Connectors";

// Why a repository may be missing: the list is the person's own GitHub view, wherever the App is installed.
export const RepositoryInstallHint = () => {
  const { data: app } = useFetchConnectorAppConfig("github");
  const installURL = app?.app_slug ? githubAppInstallURL(app) : undefined;

  return (
    <div className="space-y-1 text-xs text-muted-foreground">
      <p>
        Don't see a repository? This lists what your own GitHub account can open wherever Nexul's GitHub App is
        installed.
        {installURL && (
          <>
            {" "}
            <a href={installURL} target="_blank" rel="noreferrer" className="underline underline-offset-2 hover:text-foreground">
              Install it on an account or organisation you manage
            </a>
          </>
        )}
      </p>
      <p>
        A repository in someone else's account appears once they install the App there and give your GitHub account
        access to it. Attaching it to a project lets this workspace's deploys read it.
      </p>
    </div>
  );
};
