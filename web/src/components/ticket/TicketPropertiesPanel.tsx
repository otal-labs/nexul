import type { ReactNode } from "react";

import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { DevelopmentSection } from "@/components/ticket/DevelopmentSection";
import { TicketLabelsRow } from "@/components/ticket/TicketLabelsRow";
import { TicketPersonRow } from "@/components/ticket/TicketPersonRow";
import { TicketReporterRow } from "@/components/ticket/TicketReporterRow";
import { TicketStatusRow } from "@/components/ticket/TicketStatusRow";
import { TicketTypeRow } from "@/components/ticket/TicketTypeRow";
import { TicketRole, type Ticket, type TicketStatus as TicketStatusType } from "@/models/Ticket";

interface TicketPropertiesPanelProps {
  ticket: Ticket;
  /** Extra rail sections rendered below Attachments (e.g. the ticket page's review list). */
  children?: ReactNode;
  onTransition?: (status: TicketStatusType) => Promise<void> | void;
  onSetType?: (ticketId: string, typeId: string) => Promise<void> | void;
  onAddLabel?: (ticketId: string, label: string) => Promise<void> | void;
  onRemoveLabel?: (ticketId: string, label: string) => Promise<void> | void;
}

// Grouped-rail microheader (matches DocToc/SettingsNav/ReviewPanel's eyebrow convention).
const microheaderClass =
  "px-2 pb-1 font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

// Rows fall back to a static display when their handler is missing (read-only / no permission).
// The ticket key already names the project, so the rail has no project row; moving lives in the board and MCP.
export const TicketPropertiesPanel = ({
  ticket,
  children,
  onTransition,
  onSetType,
  onAddLabel,
  onRemoveLabel,
}: TicketPropertiesPanelProps) => (
  <aside className="hidden self-start space-y-5 text-sm lg:sticky lg:top-4 lg:block">
    <section className="space-y-0.5">
      <h2 className={microheaderClass}>Properties</h2>
      <div className="flex flex-col">
        <TicketStatusRow ticket={ticket} {...(onTransition ? { onTransition } : {})} />
        <TicketPersonRow ticket={ticket} role={TicketRole.Developer} />
        <TicketPersonRow ticket={ticket} role={TicketRole.Tester} />
        <TicketReporterRow ticket={ticket} />
        <TicketLabelsRow
          ticket={ticket}
          {...(onAddLabel ? { onAddLabel } : {})}
          {...(onRemoveLabel ? { onRemoveLabel } : {})}
        />
        <TicketTypeRow ticket={ticket} {...(onSetType ? { onSetType } : {})} />
      </div>
    </section>
    <DevelopmentSection ticketId={ticket.id} />
    <AttachmentsSection owner={{ ticket_id: ticket.id }} className="px-2" />
    {children}
  </aside>
);
