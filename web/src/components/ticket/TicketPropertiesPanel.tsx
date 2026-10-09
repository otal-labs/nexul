import type { ReactNode } from "react";

import { microheaderClass } from "@/components/Microheader";
import { TicketCategoryRow } from "@/components/ticket/TicketCategoryRow";
import { TicketLabelsRow } from "@/components/ticket/TicketLabelsRow";
import { TicketPersonRow } from "@/components/ticket/TicketPersonRow";
import { TicketReporterRow } from "@/components/ticket/TicketReporterRow";
import { TicketStatusRow } from "@/components/ticket/TicketStatusRow";
import { TicketTypeRow } from "@/components/ticket/TicketTypeRow";
import { TicketRole, type Ticket } from "@/models/Ticket";
import { cn } from "@/lib/utils";

interface TicketPropertiesPanelProps {
  ticket: Ticket;
  /** The rail sections below Properties, in order. */
  children?: ReactNode;
  onTransition?: (statusId: string) => Promise<void> | void;
  onSetType?: (ticketId: string, typeId: string) => Promise<void> | void;
  onAddLabel?: (ticketId: string, label: string) => Promise<void> | void;
  onRemoveLabel?: (ticketId: string, label: string) => Promise<void> | void;
}

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
  <aside className="min-w-0 space-y-5 self-start text-sm lg:sticky lg:top-4 lg:max-h-[calc(100dvh-2rem)] lg:overflow-y-auto">
    <section className="space-y-0.5">
      <h2 className={cn(microheaderClass, "px-2 pb-1")}>Properties</h2>
      <div className="flex flex-col">
        <TicketStatusRow ticket={ticket} {...(onTransition ? { onTransition } : {})} />
        <TicketCategoryRow ticket={ticket} />
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
    {children}
  </aside>
);
