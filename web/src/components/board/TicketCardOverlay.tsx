import { TicketCardBody } from "@/components/board/TicketCard";

import type { Ticket } from "@/models/Ticket";

interface TicketCardOverlayProps {
  ticket: Ticket;
}

// Lives in dnd-kit's portalled <DragOverlay> so column overflow never clips it; lifted to 1.03, set down by the drop animation.
export const TicketCardOverlay = ({ ticket }: TicketCardOverlayProps) => (
  <div className="relative flex cursor-grabbing select-none flex-col gap-2.5 rounded-lg border border-border bg-card p-3 transition-[scale] duration-200 ease-out motion-safe:scale-[1.03] motion-safe:animate-[lift_150ms_var(--ease-out)]">
    <span data-lift-shadow aria-hidden className="pointer-events-none absolute inset-0 rounded-[inherit] shadow-elevated transition-opacity duration-200 ease-out" />
    <TicketCardBody ticket={ticket} />
  </div>
);
