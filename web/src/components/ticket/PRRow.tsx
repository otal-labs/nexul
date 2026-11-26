import { PRChip } from "@/components/ticket/PRChip";
import { UnlinkPRButton } from "@/components/ticket/UnlinkPRButton";
import type { PRLink, PRLinkState } from "@/models/Ticket";

const stateWords: Record<PRLinkState, string> = { open: "Open", merged: "Merged", closed: "Closed" };

interface PRRowProps {
  ticketId: string;
  pr: PRLink;
}

// The title on one line, the mark, number and state in words under it.
export const PRRow = ({ ticketId, pr }: PRRowProps) => (
  <li className="group flex min-w-0 items-center gap-2 text-xs">
    <span className="min-w-0 flex-1">
      <span dir="auto" className="block truncate text-foreground" title={pr.title}>
        {pr.title}
      </span>
      <span className="mt-0.5 flex items-center gap-2 text-muted-foreground">
        <PRChip pr={pr} />
        <span>{stateWords[pr.state]}</span>
      </span>
    </span>
    <UnlinkPRButton ticketId={ticketId} pr={pr} />
  </li>
);
