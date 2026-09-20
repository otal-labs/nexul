import { useSortable } from "@dnd-kit/sortable";
import { LoaderCircle, MessageSquare } from "lucide-react";
import { memo, useRef, type CSSProperties } from "react";
import { useNavigate } from "react-router";

import { AssigneeAvatar } from "@/components/AssigneeAvatar";
import type { DropTargetData } from "@/components/board/dragMove";
import { labelDotColor, pillClass, ticketTypeColor } from "@/components/board/ticketTypeColor";
import { ticketTypeIcon } from "@/components/board/ticketTypeIcon";
import { useFetchChatThreadIndicators } from "@/hooks/ChatHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchLabelColors } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import { useIsTicketRunActive } from "@/hooks/TrailHooks";
import { cn } from "@/lib/utils";
import { ticketPath, type Ticket } from "@/models/Ticket";

interface TicketCardProps {
  ticket: Ticket;
  /** Position within its column — drives the mount stagger. */
  index?: number;
}

// Reflow/entrance stagger; matches the `window.matchMedia?.(...) ?? false` reduced-motion idiom used elsewhere.
const STAGGER_STEP_MS = 24;
const STAGGER_MAX_INDEX = 7;

interface TicketCardBodyProps {
  ticket: Ticket;
}

// Shared with TicketCardOverlay; memoized because dnd-kit re-renders every sortable on each pointer move.
export const TicketCardBody = memo(({ ticket }: TicketCardBodyProps) => {
  // All four queries are cached/deduped across every mounted TicketCard, so a column of 50 fires one request each.
  const { data: project } = useFetchProject(ticket.project_id);
  const { data: ticketTypes } = useFetchProjectTicketTypes(ticket.project_id);
  const { data: labelColors } = useFetchLabelColors(ticket.project_id);
  const { data: threadIndicators } = useFetchChatThreadIndicators(ticket.project_id);
  const hasThread = threadIndicators?.[ticket.id] === true;
  const runActive = useIsTicketRunActive(ticket.project_id, ticket.id);
  const prefix = project?.prefix ?? "";
  const ticketType = ticketTypes?.find((t) => t.id === ticket.type_id);
  const type = ticketType?.name ?? "";
  const TypeIcon = ticketTypeIcon(type);
  const labels = ticket.labels ?? [];

  return (
    <>
      {/* The whole card is the drag handle and click target; a still click never activates dnd-kit, so no inner handler is needed. */}
      <span className="flex items-center gap-2.5">
        {ticket.assignee && <AssigneeAvatar login={ticket.assignee} className="size-7 text-[10px]" />}
        <span className="min-w-0 flex-1 text-sm font-medium leading-snug">{ticket.title}</span>
        {hasThread && (
          <MessageSquare className="size-3.5 shrink-0 text-muted-foreground" role="img" aria-label="Has a chat thread" />
        )}
      </span>
      {/* Tinted pills for type and labels, ticket id on the right; color stays inside the pills, never on the card. */}
      <span className="flex items-end justify-between gap-2">
        <span className="flex min-w-0 flex-wrap items-center gap-1.5">
          {type !== "" && (
            <span
              data-slot="pill"
              className={cn("inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium", pillClass(ticketTypeColor(type, ticketType?.color)))}
            >
              <TypeIcon className="size-3" aria-hidden />
              {type}
            </span>
          )}
          {labels.map((label) => (
            <span
              key={label}
              data-slot="pill"
              className={cn("inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium", pillClass(labelDotColor(label, labelColors?.[label])))}
            >
              {label}
            </span>
          ))}
        </span>
        <span className="flex shrink-0 items-center gap-1.5 font-mono text-[10.5px] text-muted-foreground">
          {runActive && (
            <LoaderCircle className="size-3 shrink-0 animate-spin text-warning" role="img" aria-label="A play is running" />
          )}
          {prefix}-{ticket.number}
        </span>
      </span>
    </>
  );
});

const TicketCardImpl = ({ ticket, index = 0 }: TicketCardProps) => {
  const navigate = useNavigate();
  // Same cached query as TicketCardBody below — one request per board, not per card.
  const { data: project } = useFetchProject(ticket.project_id);
  const path = ticketPath(ticket, project?.prefix);
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: ticket.id,
    data: { type: "card", ticketId: ticket.id, statusId: ticket.status, categoryId: ticket.category_id } satisfies DropTargetData,
    transition: { duration: 200, easing: "ease" },
  });
  const reduceMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
  // The ghost remounts in every column it's dragged through; replaying the staggered entrance there reads as lag.
  const mountedWhileDragging = useRef(isDragging);
  const entrance = !reduceMotion && !mountedWhileDragging.current;

  // Explicit inline animation-* longhands so the mount animation never fights the permanent hover transition.
  const entranceStyle: CSSProperties = !entrance
    ? {}
    : {
        animationDelay: `${Math.min(index, STAGGER_MAX_INDEX) * STAGGER_STEP_MS}ms`,
        animationDuration: "200ms",
        animationTimingFunction: "var(--ease-out)",
      };

  // dnd-kit's inline `transition` fully replaces the className's, which would kill hover transitions, so append.
  const dragTransition = transition
    ? `${transition}, opacity 150ms var(--ease-standard), border-color 150ms var(--ease-standard)`
    : undefined;

  // Equivalent to dnd-kit's CSS.Transform.toString: translate3d plus per-axis scale while dragging.
  const dragCssTransform = transform
    ? `translate3d(${Math.round(transform.x)}px, ${Math.round(transform.y)}px, 0) scaleX(${transform.scaleX}) scaleY(${transform.scaleY})`
    : undefined;

  return (
    <div
      ref={setNodeRef}
      style={{ ...entranceStyle, transform: dragCssTransform, transition: dragTransition }}
      {...listeners}
      {...attributes}
      onClick={() => navigate(path)}
      onKeyDown={(event) => {
        // Space is dnd-kit's keyboard drag pickup; only Enter opens the ticket, matching native <button>.
        if (event.key === "Enter") navigate(path);
      }}
      className={cn(
        "group flex select-none flex-col gap-2.5 rounded-lg border border-border bg-card p-3 transition-[opacity,border-color] duration-150 ease-standard hover:border-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50",
        "cursor-grab active:cursor-grabbing",
        entrance && "animate-in fade-in-0 slide-in-from-bottom-1 fill-mode-both",
        // TicketCardOverlay carries the "lifted" look; this is just a dimmed placeholder for the slot.
        isDragging && "opacity-40",
      )}
    >
      <TicketCardBody ticket={ticket} />
    </div>
  );
};

// Memoized so a ghost entering one column doesn't re-render every card on the board.
export const TicketCard = memo(TicketCardImpl);
