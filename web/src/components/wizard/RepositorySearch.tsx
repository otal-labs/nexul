import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ConnectGitHubPrompt } from "@/components/github/ConnectGitHubPrompt";
import { RepositorySearchField, type RepositorySearchProps } from "@/components/wizard/RepositorySearchField";
import { useFetchGitHubLink } from "@/hooks/GitHubLinkHooks";

// Repositories list only through the person's own GitHub sign-in, so without one the search is a Connect GitHub prompt.
export const RepositorySearch = (props: RepositorySearchProps) => {
  const { data: link, error, isPending } = useFetchGitHubLink();

  return (
    <div>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Couldn't check your GitHub connection." />}
      {link && link.state !== "connected" && <ConnectGitHubPrompt state={link.state} />}
      {link?.state === "connected" && <RepositorySearchField {...props} />}
    </div>
  );
};
