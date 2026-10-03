import { GitBranchIcon } from "lucide-react";

import type { BranchLink } from "@/models/Ticket";

interface BranchRowProps {
  branch: BranchLink;
}

export const BranchRow = ({ branch }: BranchRowProps) => (
  <li>
    <a
      href={`https://github.com/${branch.owner}/${branch.repo}/tree/${branch.branch}`}
      target="_blank"
      rel="noreferrer"
      className="flex items-center gap-1.5 text-muted-foreground hover:text-foreground"
    >
      <GitBranchIcon className="size-3 shrink-0" aria-hidden />
      <span className="min-w-0 truncate font-mono text-xs">{branch.branch}</span>
    </a>
  </li>
);
