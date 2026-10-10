import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ConnectGitHubPrompt } from "@/components/github/ConnectGitHubPrompt";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { GitHubInstallationRow } from "@/components/settings/GitHubInstallationRow";
import { useFetchInstallations } from "@/hooks/RepositoryHooks";
import { githubLinkRefusal } from "@/models/GitHubLink";

// Read with the viewer's own GitHub sign-in; a token GitHub stopped refreshing asks for a reconnect instead of an error.
export const GitHubInstallationsList = () => {
  const { data: installations, error, isPending } = useFetchInstallations();
  const refusal = githubLinkRefusal(error);

  return (
    <>
      {isPending && <LoadingDisplay className="p-4" />}
      {refusal && <ConnectGitHubPrompt state={refusal} />}
      {error && !refusal && <ErrorDisplay error={error} title="Couldn't load the accounts." className="p-4" />}
      {installations && installations.length === 0 && (
        <EmptyRow>The GitHub App isn't installed on any account your GitHub can open yet.</EmptyRow>
      )}
      {installations && installations.length > 0 && (
        <EnterList className="divide-y divide-border overflow-hidden rounded-md border border-border">
          {installations.map((installation) => (
            <GitHubInstallationRow key={installation.account_id} installation={installation} />
          ))}
        </EnterList>
      )}
    </>
  );
};
