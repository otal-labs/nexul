import { SearchIcon } from "lucide-react";
import type { ReactNode } from "react";

import { Input } from "@/components/ui/input";
import { RepositoryRow } from "@/components/wizard/RepositoryRow";
import type { Repo } from "@/models/Repository";

import { REPOS, SEARCH_TEXT } from "./mock";

interface SearchMockProps {
  onPick: (repo: Repo) => void;
  busyId?: number | undefined;
  // Renders a repository's row itself instead of the stock one (the picked row in "On the row").
  renderRow?: (repo: Repo) => ReactNode;
  // Under the list, before the install hint.
  after?: ReactNode;
  hideList?: boolean;
}

export const SearchField = () => (
  <div className="relative">
    <SearchIcon className="absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
    <Input defaultValue={SEARCH_TEXT} placeholder="Search repositories…" aria-label="Search repositories" className="px-8" />
  </div>
);

export const InstallHint = () => (
  <p className="text-xs text-muted-foreground">
    Don't see a repository? This lists what your own GitHub account can open wherever Nexul's GitHub App is installed.
  </p>
);

export const SearchMock = ({ onPick, busyId, renderRow, after, hideList = false }: SearchMockProps) => (
  <div className="space-y-4">
    <SearchField />
    {!hideList && (
      <ul className="divide-y divide-border border-y border-border">
        {REPOS.map((repo) => (
          <SearchRow key={repo.id} repo={repo} busyId={busyId} onPick={onPick} renderRow={renderRow} />
        ))}
      </ul>
    )}
    {after}
    <InstallHint />
  </div>
);

interface SearchRowProps {
  repo: Repo;
  busyId?: number | undefined;
  onPick: (repo: Repo) => void;
  renderRow?: ((repo: Repo) => ReactNode) | undefined;
}

const SearchRow = ({ repo, busyId, onPick, renderRow }: SearchRowProps) => {
  const custom = renderRow?.(repo);
  return (
    <>
      {custom}
      {!custom && <RepositoryRow repo={repo} busy={busyId === repo.id} onSelect={() => onPick(repo)} />}
    </>
  );
};
