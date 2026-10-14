import { BookMarkedIcon } from "lucide-react";

import { BranchRow } from "@/components/ticket/BranchRow";
import { PRChip } from "@/components/ticket/PRChip";
import { UnlinkBranchButton } from "@/components/ticket/UnlinkBranchButton";
import { UnlinkPRButton } from "@/components/ticket/UnlinkPRButton";
import type { RepoLinks } from "@/utils/TicketLinksUtility";

interface RepoLinksRowProps {
  ticketId: string;
  links: RepoLinks;
}

export const RepoLinksRow = ({ ticketId, links }: RepoLinksRowProps) => (
  <li className="flex flex-col gap-1 px-2 py-1">
    <a
      href={`https://github.com/${links.owner}/${links.repo}`}
      target="_blank"
      rel="noreferrer"
      className="flex min-w-0 items-center gap-2 hover:underline"
    >
      <BookMarkedIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
      <span className="truncate font-mono text-xs">
        {links.owner}/{links.repo}
      </span>
    </a>
    {links.branches.length > 0 && (
      <ul className="flex flex-col gap-0.5 pl-5.5">
        {links.branches.map((branch) => (
          <BranchRow key={branch.branch} branch={branch}>
            <UnlinkBranchButton ticketId={ticketId} branch={branch} />
          </BranchRow>
        ))}
      </ul>
    )}
    {links.prs.length > 0 && (
      <div className="flex flex-wrap gap-x-3 gap-y-1 pl-5.5">
        {links.prs.map((pr) => (
          <span key={pr.number} className="group inline-flex items-center gap-0.5">
            <PRChip pr={pr} />
            <UnlinkPRButton ticketId={ticketId} pr={pr} />
          </span>
        ))}
      </div>
    )}
  </li>
);
