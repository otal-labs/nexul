import { microheaderClass } from "@/components/Microheader";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { AddTicketLinkMenu } from "@/components/ticket/AddTicketLinkMenu";
import { LinkGroupSection } from "@/components/ticket/LinkGroupSection";
import { LinkedTicketRow } from "@/components/ticket/LinkedTicketRow";
import { OriginUnknownRow } from "@/components/ticket/OriginUnknownRow";
import { TicketSourceRow } from "@/components/ticket/TicketSourceRow";
import { useSetTicketSource } from "@/hooks/TicketHooks";
import { useFetchTicketLinkSet, useRemoveBlocker, useRemoveFoundIn } from "@/hooks/TicketLinkHooks";
import type { Ticket } from "@/models/Ticket";

interface TicketLinksSectionProps {
  ticket: Ticket;
}

// The source doc, blockers, and the found-in origin; the bugs found in this ticket are their own rail section.
export const TicketLinksSection = ({ ticket }: TicketLinksSectionProps) => {
  const ticketId = ticket.id;
  const { data: links, error, isPending } = useFetchTicketLinkSet(ticketId);
  const removeBlocker = useRemoveBlocker();
  const removeFoundIn = useRemoveFoundIn();
  const setSource = useSetTicketSource();
  const empty = links && !ticket.doc_id && !links.found_in && !links.origin_unknown && links.blocked_by.length + links.blocks.length === 0;

  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between px-2">
        <h2 className={microheaderClass}>Links</h2>
        <AddTicketLinkMenu ticket={ticket} />
      </div>
      {isPending && <LoadingDisplay label="Loading linked tickets…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load linked tickets." />}
      {empty && <p className="px-2 text-xs text-muted-foreground">No links yet.</p>}
      {ticket.doc_id && (
        <LinkGroupSection title="Source">
          <TicketSourceRow docId={ticket.doc_id} onRemove={() => setSource.mutate({ id: ticketId, docId: "" })} />
        </LinkGroupSection>
      )}
      {links && links.blocked_by.length > 0 && (
        <LinkGroupSection title={links.blocked ? "Blocked by" : "Blocked by (all done)"}>
          {links.blocked_by.map((t) => (
            <LinkedTicketRow
              key={t.id}
              ticket={t}
              waiting
              onRemove={() => removeBlocker.mutate({ id: ticketId, blockerId: t.id })}
            />
          ))}
        </LinkGroupSection>
      )}
      {links && links.blocks.length > 0 && (
        <LinkGroupSection title="Blocks">
          {links.blocks.map((t) => (
            <LinkedTicketRow
              key={t.id}
              ticket={t}
              onRemove={() => removeBlocker.mutate({ id: t.id, blockerId: ticketId })}
            />
          ))}
        </LinkGroupSection>
      )}
      {links?.found_in && (
        <LinkGroupSection title="Found in">
          <LinkedTicketRow ticket={links.found_in} onRemove={() => removeFoundIn.mutate({ id: ticketId })} />
        </LinkGroupSection>
      )}
      {links?.origin_unknown && (
        <LinkGroupSection title="Found in">
          <OriginUnknownRow onRemove={() => removeFoundIn.mutate({ id: ticketId })} />
        </LinkGroupSection>
      )}
    </section>
  );
};
