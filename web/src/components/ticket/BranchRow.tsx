import { GitBranchIcon } from "lucide-react";

import type { BranchLink } from "@/models/Ticket";

interface BranchRowProps {
  branch: BranchLink;
}

// One-line row, no card/border of its own — the list it sits in owns the shape.
export const BranchRow = ({ branch }: BranchRowProps) => (
  <li className="flex items-center gap-2 rounded-md px-2 py-1 transition-colors duration-150 ease-standard hover:bg-muted/50">
    <GitBranchIcon className="size-3.5 shrink-0 text-muted-foreground" />
    <span className="min-w-0 flex-1 truncate font-mono text-xs">
      {branch.owner}/{branch.repo}:{branch.branch}
    </span>
  </li>
);
