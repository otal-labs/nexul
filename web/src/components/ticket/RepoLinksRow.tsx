import { BookMarkedIcon } from "lucide-react";

import { BranchRow } from "@/components/ticket/BranchRow";
import { PRRow } from "@/components/ticket/PRRow";
import { UnlinkBranchButton } from "@/components/ticket/UnlinkBranchButton";
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
      <ul className="flex flex-col gap-1 pl-5.5">
        {links.prs.map((pr) => (
          <PRRow key={pr.number} ticketId={ticketId} pr={pr} />
        ))}
      </ul>
    )}
  </li>
);
