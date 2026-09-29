import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { GitHubInstallationRow } from "@/components/settings/GitHubInstallationRow";
import { useFetchInstallations } from "@/hooks/RepositoryHooks";

// Reads through the GitHub connector's token, so only mount it while that connector is connected.
export const GitHubInstallationsList = () => {
  const { data: installations, error, isPending } = useFetchInstallations();

  return (
    <>
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
    </>
  );
};
