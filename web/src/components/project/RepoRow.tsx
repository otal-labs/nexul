import { XIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useRemoveProjectRepo } from "@/hooks/ProjectHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { RepoRole } from "@/enums/Project";
import type { RepoRef } from "@/models/Project";

interface RepoRowProps {
  repo: RepoRef;
}

export const RepoRow = ({ repo }: RepoRowProps) => {
  const removeRepo = useRemoveProjectRepo();
  const { open: confirm } = useConfirmationDialog();

  const remove = async () => {
    const ok = await confirm({
      title: `Remove ${repo.full_name}?`,
      message: "Its pull requests stop linking to this project's tickets. The repository itself isn't touched, and you can add it again.",
      confirmLabel: "Remove repo",
    });
    if (ok) removeRepo.mutate({ owner: repo.owner, name: repo.name });
  };

  return (
    <li className="flex items-center gap-3 px-3 py-2 text-sm transition-colors duration-150 ease-standard hover:bg-accent/40">
      <span className="flex min-w-0 flex-1 items-center gap-2">
        <span className="truncate font-mono text-xs">{repo.full_name}</span>
        {repo.role === RepoRole.Tests && (
          <span className="shrink-0 font-mono text-xs text-muted-foreground" title="Tests repository, never deployed">
            tests
          </span>
        )}
      </span>
      <span className="shrink-0 font-mono text-xs text-muted-foreground">{repo.connector_id}</span>
      <Button
        variant="ghost"
        size="icon"
        className="size-8 shrink-0 hover:text-destructive"
        aria-label={`Remove ${repo.full_name}`}
        title={`Remove ${repo.full_name}`}
        loading={removeRepo.isPending}
        onClick={() => void remove()}
      >
        <XIcon className="size-4" />
      </Button>
    </li>
  );
};
