import { Bug } from "lucide-react";

import { Button } from "@/components/ui/button";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { AddTicketLinkMenu } from "@/components/ticket/AddTicketLinkMenu";
import { LinkGroupSection } from "@/components/ticket/LinkGroupSection";
import { LinkedTicketRow } from "@/components/ticket/LinkedTicketRow";
import { OriginUnknownRow } from "@/components/ticket/OriginUnknownRow";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTicketLinkSet, useRemoveBlocker, useRemoveFoundIn } from "@/hooks/TicketLinkHooks";
import { useReportBugDialog } from "@/hooks/useReportBugDialog";
import { StatusKind } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

const microheaderClass =
  "px-2 font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

interface TicketLinksSectionProps {
  ticket: Ticket;
}

// Sits in the main column so it shows at every width; the properties rail is desktop-only.
export const TicketLinksSection = ({ ticket }: TicketLinksSectionProps) => {
  const ticketId = ticket.id;
  const { data: links, error, isPending } = useFetchTicketLinkSet(ticketId);
  const { data: statuses } = useFetchProjectStatuses(ticket.project_id);
  const removeBlocker = useRemoveBlocker();
  const removeFoundIn = useRemoveFoundIn();
  const reportBug = useReportBugDialog();
  // A done ticket is never reopened (ADR 0064), so every bug linked to it was found after done.
  const done = statuses?.some((s) => s.id === ticket.status && s.kind === StatusKind.Done) ?? false;
  const empty =
    links &&
    !links.found_in &&
    !links.origin_unknown &&
    links.blocked_by.length + links.blocks.length + links.bugs_found.length === 0;

  return (
    <section className="space-y-3 border-t border-border pt-6">
      <div className="flex items-center justify-between">
        <h2 className={microheaderClass}>Linked tickets</h2>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-xs text-muted-foreground"
            onClick={() => void reportBug({ originId: ticketId, projectId: ticket.project_id })}
          >
            <Bug className="size-3.5" aria-hidden />
            Report a bug
          </Button>
          <AddTicketLinkMenu ticketId={ticketId} />
        </div>
      </div>
      {isPending && <LoadingDisplay label="Loading linked tickets…" />}
      {error && <ErrorDisplay error={error} title="Failed to load linked tickets." />}
      {empty && <EmptyRow className="py-4">No blockers or found-in links.</EmptyRow>}
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
      {links && links.bugs_found.length > 0 && (
        <LinkGroupSection title={done ? "Bugs found after done" : "Bugs found in this"}>
          {links.bugs_found.map((t) => (
            <LinkedTicketRow key={t.id} ticket={t} />
          ))}
        </LinkGroupSection>
      )}
    </section>
  );
};
