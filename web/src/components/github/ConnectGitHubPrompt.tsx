import { EmptyState } from "@/components/EmptyState";
import { GithubMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";
import { useStartIdentityLink } from "@/hooks/AuthHooks";
import type { GitHubLinkState } from "@/models/GitHubLink";

interface ConnectGitHubPromptProps {
  state: Exclude<GitHubLinkState, "connected">;
}

// Where a list of repositories would be: Nexul reads them only with the person's own GitHub sign-in.
export const ConnectGitHubPrompt = ({ state }: ConnectGitHubPromptProps) => {
  const link = useStartIdentityLink();
  const verb = state === "reconnect" ? "Reconnect" : "Connect";

  return (
    <EmptyState
      icon={GithubMark}
      size="compact"
      title={`${verb} GitHub to see your repositories`}
      message={
        state === "reconnect"
          ? "GitHub asked for your account to be connected again. Nexul lists only what your own GitHub account can open."
          : "Nexul lists only the repositories your own GitHub account can open where its GitHub App is installed."
      }
      action={
        <Button variant="outline" size="sm" loading={link.isPending} onClick={() => link.mutate("github")}>
          {verb} GitHub
        </Button>
      }
    />
  );
};
