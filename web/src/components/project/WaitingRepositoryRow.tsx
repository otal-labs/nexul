import { Rocket } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { RepoRef } from "@/models/Project";

interface WaitingRepositoryRowProps {
  projectId: string;
  repo: RepoRef;
}

// An attached repository no service builds from yet, with the way into the wizard's Service step for it.
export const WaitingRepositoryRow = ({ projectId, repo }: WaitingRepositoryRowProps) => {
  const wsPath = useWorkspacePath();
  const to = wsPath(`/wizard/project/repository?project=${projectId}&add=1&repo=${encodeURIComponent(repo.full_name)}`);
  return (
    <li className="flex items-center gap-3 px-3 py-2.5">
      <span className="min-w-0 flex-1">
        <span className="block truncate font-mono text-xs">{repo.full_name}</span>
        <span className="mt-0.5 flex items-center gap-1.5 text-xs text-muted-foreground">
          <span aria-hidden className="size-1.5 shrink-0 rounded-full border border-muted-foreground" />
          Attached, not deployed yet
        </span>
      </span>
      <Button asChild variant="outline" size="sm" className="shrink-0">
        <Link to={to}>
          <Rocket className="size-3.5" aria-hidden />
          Deploy this repository
        </Link>
      </Button>
    </li>
  );
};
