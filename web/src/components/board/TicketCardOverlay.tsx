import { useDndContext } from "@dnd-kit/core";
import { useEffect, useRef } from "react";

import { followTilt } from "@/components/board/dragTilt";
import { TicketCardBody } from "@/components/board/TicketCard";

import type { Ticket } from "@/models/Ticket";

interface TicketCardOverlayProps {
  ticket: Ticket;
}

// Lives in dnd-kit's <DragOverlay>, portalled to body so neither column overflow nor the page frame shifts it; lifted to 1.03, set down by the drop animation.
export const TicketCardOverlay = ({ ticket }: TicketCardOverlayProps) => {
  const ref = useRef<HTMLDivElement>(null);
  const { activatorEvent, activeNodeRect } = useDndContext();
  // Read once: the source card re-measures as the preview moves it, and the tilt must not restart.
  const start = useRef({ activatorEvent, activeNodeRect });
  useEffect(() => followTilt(ref.current, start.current.activatorEvent, start.current.activeNodeRect), []);

  return (
    <div
      ref={ref}
      className="relative flex cursor-grabbing select-none flex-col gap-1.5 rounded-lg border border-border bg-card px-2.5 py-2 transition-[scale] duration-200 ease-out motion-safe:scale-[1.03] motion-safe:animate-[lift_150ms_var(--ease-out)]"
    >
      <span data-lift-shadow aria-hidden className="pointer-events-none absolute inset-0 rounded-[inherit] shadow-elevated transition-opacity duration-200 ease-out" />
      <TicketCardBody ticket={ticket} />
    </div>
  );
};
