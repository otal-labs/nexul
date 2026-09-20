import { GitMergeIcon, GitPullRequestClosedIcon, GitPullRequestIcon } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import type { PRLink } from "@/models/Ticket";

const PRStateLabel = {
  open: "open",
  merged: "merged",
  closed: "closed",
} as const;

const stateIcons: Record<keyof typeof PRStateLabel, LucideIcon> = {
  open: GitPullRequestIcon,
  merged: GitMergeIcon,
  closed: GitPullRequestClosedIcon,
};

const stateColors: Record<keyof typeof PRStateLabel, string> = {
  open: "text-info",
  merged: "text-success",
  closed: "text-muted-foreground",
};

interface PRRowProps {
  pr: PRLink;
}

// One-line row with state badge, no card/border of its own — the list owns the shape.
export const PRRow = ({ pr }: PRRowProps) => (
  <li className="flex items-center gap-2 rounded-md px-2 py-1 transition-colors duration-150 ease-standard hover:bg-muted/50">
    <GitPullRequestIcon className="size-3.5 shrink-0 text-muted-foreground" />
    <span className="min-w-0 flex-1 truncate font-mono text-xs">
      {pr.owner}/{pr.repo}#{pr.number}
    </span>
    <NoFillBadge icon={stateIcons[pr.state]} color={stateColors[pr.state]}>
      {PRStateLabel[pr.state]}
    </NoFillBadge>
  </li>
);
