import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { GitHubAccessRow } from "@/components/you/GitHubAccessRow";
import { useBootstrapStatus } from "@/hooks/AuthHooks";
import { useFetchGitHubLink } from "@/hooks/GitHubLinkHooks";

// Shown once GitHub sign-in is on: the project wizard lists repositories only through this link.
export const GitHubAccessSection = () => {
  const { data: status } = useBootstrapStatus();
  const { data: link, error, isPending } = useFetchGitHubLink();
  if (!status?.configured) return null;

  return (
    <SettingsCard
      id="github-access"
      title="GitHub"
      description="Nexul lists the repositories your own GitHub account can open where its GitHub App is installed, and nothing else. Disconnecting stops the list until you connect again."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {link && (
        <ul className="overflow-hidden rounded-md border border-border">
          <GitHubAccessRow link={link} />
        </ul>
      )}
    </SettingsCard>
  );
};
