import { TicketCardBody } from "@/components/board/TicketCard";
import type { Ticket } from "@/models/Ticket";

interface TicketCardOverlayProps {
  ticket: Ticket;
}

// Lives in dnd-kit's portalled <DragOverlay> so column overflow never clips it; KanbanBoard's drop animation levels the tilt.
export const TicketCardOverlay = ({ ticket }: TicketCardOverlayProps) => (
  <div className="flex rotate-3 cursor-grabbing select-none flex-col gap-2.5 rounded-lg border border-border bg-card p-3 shadow-elevated transition-[rotate] duration-200 ease-out">
    <TicketCardBody ticket={ticket} />
  </div>
);
