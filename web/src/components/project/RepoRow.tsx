import { XIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useRemoveProjectRepo } from "@/hooks/ProjectHooks";
import { RepoRole } from "@/enums/Project";
import type { RepoRef } from "@/models/Project";

interface RepoRowProps {
  repo: RepoRef;
}

export const RepoRow = ({ repo }: RepoRowProps) => {
  const removeRepo = useRemoveProjectRepo();

  return (
    <li className="-mx-2 flex items-center gap-2 rounded-sm px-2 py-1.5 text-sm transition-colors duration-[120ms] ease-standard hover:bg-accent/40">
      <span className="flex min-w-0 flex-1 items-center gap-2">
        <span className="truncate font-mono text-xs">{repo.full_name}</span>
        {repo.role === RepoRole.Tests && (
          <span className="shrink-0 font-mono text-xs text-muted-foreground" title="Tests repository, never deployed">
            tests
          </span>
        )}
      </span>
      <span className="w-20 shrink-0 truncate font-mono text-xs text-muted-foreground">{repo.connector_id}</span>
      <Button
        variant="ghost"
        size="sm"
        className="w-8 shrink-0"
        aria-label={`Remove ${repo.full_name}`}
        onClick={() => removeRepo.mutate({ owner: repo.owner, name: repo.name })}
      >
        <XIcon className="size-4" />
      </Button>
    </li>
  );
};
