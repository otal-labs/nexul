import { PRChip } from "@/components/ticket/PRChip";
import { UnlinkPRButton } from "@/components/ticket/UnlinkPRButton";
import type { PRLink, PRLinkState } from "@/models/Ticket";

const stateWords: Record<PRLinkState, string> = { open: "Open", merged: "Merged", closed: "Closed" };

interface PRRowProps {
  ticketId: string;
  pr: PRLink;
}

// The mark and number, the title on up to two lines, then the state in words beside its hue.
export const PRRow = ({ ticketId, pr }: PRRowProps) => (
  <li className="group flex min-w-0 items-start gap-2 text-xs">
    <PRChip pr={pr} />
    <span dir="auto" className="min-w-0 flex-1 line-clamp-2 text-muted-foreground" title={pr.title}>
      {pr.title}
    </span>
    <span className="shrink-0 text-muted-foreground">{stateWords[pr.state]}</span>
    <span className="flex self-center">
      <UnlinkPRButton ticketId={ticketId} pr={pr} />
    </span>
  </li>
);
