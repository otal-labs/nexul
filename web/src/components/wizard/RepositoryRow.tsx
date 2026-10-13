import type { ReactNode } from "react";

import { RepoProviderMark } from "@/components/ProviderMarks";
import type { Repo } from "@/models/Repository";

interface RepositoryRowProps {
  repo: Repo;
  busy: boolean;
  onSelect: () => void;
  // The picked repository opens in place, showing this instead of the plain row.
  open?: ReactNode;
}

export const RepositoryRow = ({ repo, busy, onSelect, open }: RepositoryRowProps) => (
  <li>
    {open}
    {!open && (
      <button
        type="button"
        onClick={onSelect}
        disabled={busy}
        className="flex w-full items-center gap-3 px-2 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/40 disabled:opacity-60"
      >
        <RepoProviderMark provider={repo.provider} />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-medium">{repo.full_name}</span>
          <span className="block truncate font-mono text-xs text-muted-foreground">{repo.default_branch}</span>
        </span>
        {busy && <span className="shrink-0 text-xs text-muted-foreground">Scanning…</span>}
      </button>
    )}
  </li>
);
