import { PlusIcon } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { LinkGroupSection } from "@/components/ticket/LinkGroupSection";
import { LinkedTicketRow } from "@/components/ticket/LinkedTicketRow";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTicketLinkSet } from "@/hooks/TicketLinkHooks";
import { useReportBugDialog } from "@/hooks/useReportBugDialog";
import { StatusKind } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

const microheaderClass =
  "font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

interface TicketBugsSectionProps {
  ticket: Ticket;
}

// The bugs whose found-in link points at this ticket, and the way to report another.
export const TicketBugsSection = ({ ticket }: TicketBugsSectionProps) => {
  const { data: links, error, isPending } = useFetchTicketLinkSet(ticket.id);
  const { data: statuses } = useFetchProjectStatuses(ticket.project_id);
  const reportBug = useReportBugDialog();
  // A done ticket is never reopened (ADR 0064), so every bug linked to it was found after done.
  const done = statuses?.some((s) => s.id === ticket.status && s.kind === StatusKind.Done) ?? false;

  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between px-2">
        <h2 className={microheaderClass}>Bugs</h2>
        <button
          type="button"
          aria-label="Report a bug"
          onClick={() => void reportBug({ originId: ticket.id, projectId: ticket.project_id })}
          className="flex size-5 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-muted/50 hover:text-foreground"
        >
          <PlusIcon className="size-3.5" aria-hidden />
        </button>
      </div>
      {isPending && <LoadingDisplay label="Loading bugs…" />}
      {error && <ErrorDisplay error={error} title="Failed to load bugs." />}
      {links && links.bugs_found.length === 0 && <p className="px-2 text-xs text-muted-foreground">No bugs reported.</p>}
      {links && links.bugs_found.length > 0 && (
        <LinkGroupSection title={done ? "Found after done" : "Found in this ticket"}>
          {links.bugs_found.map((t) => (
            <LinkedTicketRow key={t.id} ticket={t} />
          ))}
        </LinkGroupSection>
      )}
    </section>
  );
};
