import { SearchIcon } from "lucide-react";
import { useState } from "react";

import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ConnectGitHubPrompt } from "@/components/github/ConnectGitHubPrompt";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { RepositoryInstallHint } from "@/components/wizard/RepositoryInstallHint";
import { RepositoryRow } from "@/components/wizard/RepositoryRow";
import { useSearchRepositories } from "@/hooks/RepositoryHooks";
import { githubLinkRefusal } from "@/models/GitHubLink";
import { REPOSITORY_SEARCH_MIN_LENGTH, type Repo } from "@/models/Repository";

export interface RepositorySearchProps {
  onSelect: (repo: Repo) => void;
  // A repository already taken elsewhere in the wizard, left out of the results.
  excludeId?: number | undefined;
  // The repository being scanned, shown as busy.
  busyId?: number | undefined;
  // Tells apart two searches on one page.
  label?: string;
}

// Nothing loads until enough letters are typed: an installation can grant thousands of repositories.
export const RepositorySearchField = ({ onSelect, excludeId, busyId, label = "Search repositories" }: RepositorySearchProps) => {
  const [text, setText] = useState("");
  const searching = text.trim().length >= REPOSITORY_SEARCH_MIN_LENGTH;
  const { data, isFetching, error } = useSearchRepositories(text);
  const repos = searching ? data?.filter((r) => r.id !== excludeId) : undefined;
  // A token GitHub stopped refreshing mid-search asks for a reconnect instead of an error.
  const refusal = searching ? githubLinkRefusal(error) : undefined;

  return (
    <div className="space-y-4">
      <div className="relative">
        <SearchIcon
          className="absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden
        />
        <Input
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Search repositories…"
          aria-label={label}
          className="px-8"
        />
        {searching && isFetching && (
          <Spinner
            aria-label="Searching"
            className="absolute top-1/2 right-2.5 size-4 -translate-y-1/2 text-muted-foreground"
          />
        )}
      </div>
      {!searching && (
        <p className="text-xs text-muted-foreground">
          Type at least {REPOSITORY_SEARCH_MIN_LENGTH} letters to search your repositories.
        </p>
      )}
      {refusal && <ConnectGitHubPrompt state={refusal} />}
      {searching && error && !refusal && <ErrorDisplay error={error} title="Couldn't load repositories." />}
      {repos?.length === 0 && (
        <EmptyState title="No repositories match" message="Try a different search." size="compact" />
      )}
      {repos && repos.length > 0 && (
        <ul className="divide-y divide-border border-y border-border">
          {repos.map((repo) => (
            <RepositoryRow key={repo.id} repo={repo} busy={busyId === repo.id} onSelect={() => onSelect(repo)} />
          ))}
        </ul>
      )}
      <RepositoryInstallHint />
    </div>
  );
};
