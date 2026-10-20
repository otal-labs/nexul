import { useFetchConnectorAppConfig } from "@/hooks/ConnectorsHooks";
import { githubAppInstallURL } from "@/models/Connectors";

// The list is the person's own GitHub view wherever the App is installed, so a missing repository usually means an account without it.
export const RepositoryInstallHint = () => {
  const { data: app } = useFetchConnectorAppConfig("github");
  const installURL = app?.app_slug ? githubAppInstallURL(app) : undefined;

  return (
    <p className="text-xs text-muted-foreground">
      Missing a repository?{" "}
      {installURL && (
        <a href={installURL} target="_blank" rel="noreferrer" className="underline underline-offset-2 hover:text-foreground">
          Install the GitHub App on its account
        </a>
      )}
      {!installURL && "The GitHub App has to be installed on its account."}
    </p>
  );
};
