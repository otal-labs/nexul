import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { AddTicketLinkMenu } from "@/components/ticket/AddTicketLinkMenu";
import { LinkGroupSection } from "@/components/ticket/LinkGroupSection";
import { LinkedTicketRow } from "@/components/ticket/LinkedTicketRow";
import { OriginUnknownRow } from "@/components/ticket/OriginUnknownRow";
import { useFetchTicketLinkSet, useRemoveBlocker, useRemoveFoundIn } from "@/hooks/TicketLinkHooks";
import type { Ticket } from "@/models/Ticket";

const microheaderClass =
  "font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

interface TicketLinksSectionProps {
  ticket: Ticket;
}

// Blockers and the found-in origin; the bugs found in this ticket are their own rail section.
export const TicketLinksSection = ({ ticket }: TicketLinksSectionProps) => {
  const ticketId = ticket.id;
  const { data: links, error, isPending } = useFetchTicketLinkSet(ticketId);
  const removeBlocker = useRemoveBlocker();
  const removeFoundIn = useRemoveFoundIn();
  const empty = links && !links.found_in && !links.origin_unknown && links.blocked_by.length + links.blocks.length === 0;

  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between px-2">
        <h2 className={microheaderClass}>Links</h2>
        <AddTicketLinkMenu ticketId={ticketId} />
      </div>
      {isPending && <LoadingDisplay label="Loading linked tickets…" />}
      {error && <ErrorDisplay error={error} title="Failed to load linked tickets." />}
      {empty && <p className="px-2 text-xs text-muted-foreground">No blockers or found-in links.</p>}
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
            <LinkedTicketRow key={t.id} ticket={t} />
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
