import { GitBranchIcon } from "lucide-react";
import type { ReactNode } from "react";

import type { BranchLink } from "@/models/Ticket";

interface BranchRowProps {
  branch: BranchLink;
  /** What sits after the branch, such as its unlink button; it shows on the row's hover and focus. */
  children?: ReactNode;
}

export const BranchRow = ({ branch, children }: BranchRowProps) => (
  <li className="group flex items-center gap-1">
    <a
      href={`https://github.com/${branch.owner}/${branch.repo}/tree/${branch.branch}`}
      target="_blank"
      rel="noreferrer"
      className="flex min-w-0 items-center gap-1.5 text-muted-foreground hover:text-foreground"
    >
      <GitBranchIcon className="size-3 shrink-0" aria-hidden />
      <span className="min-w-0 truncate font-mono text-xs">{branch.branch}</span>
    </a>
    {children}
  </li>
);
