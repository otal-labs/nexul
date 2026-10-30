import { Ban } from "lucide-react";
import { Link } from "react-router";

import { personLabel } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { PRChip } from "@/components/ticket/PRChip";
import { TicketStatusBadge } from "@/components/ticket/TicketStatusBadge";
import { useTicketPerson } from "@/hooks/PeopleHooks";
import { useTicketStatus } from "@/hooks/StatusHooks";
import { useFetchTicketLinks } from "@/hooks/TicketHooks";
import { useFetchTicketLinkSet } from "@/hooks/TicketLinkHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cardPerson, TicketRole, type Ticket } from "@/models/Ticket";
import { linkedTicketKey } from "@/models/TicketLink";

interface TicketHeaderStripProps {
  ticket: Ticket;
  ticketLabel: string;
}

const shownBlockers = 1;
const shownPRs = 2;

interface FoldedCountProps {
  keys: string[];
  noun: string;
}

const FoldedCount = ({ keys, noun }: FoldedCountProps) =>
  keys.length > 0 && (
    <span title={keys.join(", ")} className="font-mono text-xs whitespace-nowrap text-muted-foreground">
      <span aria-hidden>+{keys.length}</span>
      <span className="sr-only">
        {keys.length} more {noun}
        {keys.length === 1 ? "" : "s"}: {keys.join(", ")}
      </span>
    </span>
  );

// The ticket's standing in the board card's marks, the same on every tab: key, status, who acts next, what still
// blocks it, its pull requests. Read only; the rail edits.
export const TicketHeaderStrip = ({ ticket, ticketLabel }: TicketHeaderStripProps) => {
  const { status } = useTicketStatus(ticket);
  const ref = cardPerson(ticket, status?.kind);
  const person = useTicketPerson(ref);
  const { data: links } = useFetchTicketLinkSet(ticket.id);
  const { data: development } = useFetchTicketLinks(ticket.id);
  const wsPath = useWorkspacePath();
  const waitingOn = (links?.blocked_by ?? []).filter((t) => !t.done && !t.finished);
  const prs = development?.prs ?? [];
  const role = ref.role === TicketRole.Tester ? "Tester" : "Developer";

  return (
    <>
      <span className="font-mono text-xs whitespace-nowrap text-foreground">{ticketLabel}</span>
      <TicketStatusBadge ticket={ticket} />
      {ref.login && (
        <span className="inline-flex min-w-0 items-center gap-1.5 text-xs" title={role}>
          <PersonAvatar login={ref.login} src={person.avatar_url} className="size-4 text-[10px]" label={role} />
          <span dir="auto" className="truncate text-foreground">
            {personLabel(person)}
          </span>
        </span>
      )}
      {waitingOn.slice(0, shownBlockers).map((blocker) => (
        <Link
          key={blocker.id}
          to={wsPath(`/tickets/${linkedTicketKey(blocker)}`)}
          title={`Blocked by ${linkedTicketKey(blocker)}: ${blocker.title}`}
          aria-label={`Blocked by ${linkedTicketKey(blocker)}`}
          className="inline-flex items-center gap-1 font-mono text-xs whitespace-nowrap text-foreground underline-offset-4 hover:underline"
        >
          <Ban className="size-3 shrink-0 text-warning" aria-hidden />
          {linkedTicketKey(blocker)}
        </Link>
      ))}
      <FoldedCount keys={waitingOn.slice(shownBlockers).map(linkedTicketKey)} noun="blocker" />
      {prs.slice(0, shownPRs).map((pr) => (
        <PRChip key={`${pr.owner}/${pr.repo}#${pr.number}`} pr={pr} />
      ))}
      <FoldedCount keys={prs.slice(shownPRs).map((pr) => `#${pr.number}`)} noun="pull request" />
    </>
  );
};
