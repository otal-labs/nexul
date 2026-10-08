import { GitMergeIcon, GitPullRequestClosedIcon, GitPullRequestIcon } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import type { PRLink, PRLinkState } from "@/models/Ticket";

const stateIcons: Record<PRLinkState, LucideIcon> = {
  open: GitPullRequestIcon,
  merged: GitMergeIcon,
  closed: GitPullRequestClosedIcon,
};

const stateColors: Record<PRLinkState, string> = {
  open: "text-success",
  merged: "text-merged",
  closed: "text-destructive",
};

interface PRChipProps {
  pr: PRLink;
}

export const PRChip = ({ pr }: PRChipProps) => (
  <a
    href={`https://github.com/${pr.owner}/${pr.repo}/pull/${pr.number}`}
    target="_blank"
    rel="noreferrer"
    title={pr.title}
    aria-label={`#${pr.number} ${pr.title}, ${pr.state}`}
    className="rounded-md hover:underline"
  >
    <NoFillBadge icon={stateIcons[pr.state]} color={stateColors[pr.state]} className="font-mono tabular-nums">
      #{pr.number}
    </NoFillBadge>
  </a>
);
