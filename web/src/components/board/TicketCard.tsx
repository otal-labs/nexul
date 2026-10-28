import { useSortable } from "@dnd-kit/sortable";
import { useQueryClient } from "@tanstack/react-query";
import { CircleHelp, Clock, LoaderCircle, MessageSquare } from "lucide-react";
import { memo, useState } from "react";
import { useNavigate } from "react-router";

import { personLabel } from "@nexul/client-core/person";
import { findTicketPerson } from "@nexul/client-core/ticketPeople";

import { LabelChip } from "@/components/LabelChip";
import { PersonAvatar } from "@/components/PersonAvatar";
import type { DropTargetData } from "@/components/board/dragMove";
import { animateNearSlot } from "@/components/board/dragPreview";
import { RunTimer } from "@/components/board/RunTimer";
import { TicketBlockedLine } from "@/components/board/TicketBlockedLine";
import { ticketTypeColor } from "@/components/board/ticketTypeColor";
import { ticketTypeIcon } from "@/components/board/ticketTypeIcon";
import { NoFillBadge } from "@/components/ui/badge";
import { useBoardCardData, useCanMoveTickets } from "@/hooks/useBoardCardData";
import { getProjectKey } from "@/hooks/ProjectHooks";
import { useRunState } from "@/hooks/TrailHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import type { Project } from "@/models/Project";
import { cardPerson, ticketPath, type Ticket } from "@/models/Ticket";

interface TicketCardProps {
  ticket: Ticket;
  collapsed?: boolean;
}

interface TicketCardBodyProps {
  ticket: Ticket;
}

// Two chips beside the type fit a 220px column on one row; the rest count as +N, named in its tooltip.
const CARD_LABELS = 2;

// Shared with TicketCardOverlay; memoized because dnd-kit re-renders every sortable on each pointer move.
export const TicketCardBody = memo(({ ticket }: TicketCardBodyProps) => {
  const { prefix, ticketTypes, labelColors, threadIndicators, statuses, activeTrails, people } = useBoardCardData();
  const person = cardPerson(ticket, statuses?.find((s) => s.id === ticket.status)?.kind);
  const shown = findTicketPerson(people, person);
  const hasThread = threadIndicators?.[ticket.id] === true;
  const runState = useRunState("ticket", ticket.id, activeTrails);
  const runStartedAt = activeTrails?.started[ticket.id];
  const runQueued = activeTrails?.queued[ticket.id] !== undefined;
  const ticketType = ticketTypes?.find((t) => t.id === ticket.type_id);
  const type = ticketType?.name ?? "";
  const labels = ticket.labels ?? [];
  // A bug keeps its red; every other type's icon is muted, so the card's colour is its labels and its status, not a second accent.
  const typeIconColor = type.trim().toLowerCase() === "bug" ? ticketTypeColor(type, ticketType?.color) : "text-muted-foreground";

  const avatar = person.login && (
    <span role="img" aria-label={`${person.role} ${personLabel(shown)}`} className="shrink-0">
      <PersonAvatar login={person.login} src={shown.avatar_url} className="size-4 text-[10px]" />
    </span>
  );
  const pills = (
    <span className="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-1">
      {type !== "" && (
        // The type is its icon in its hue, named by the tooltip: the word beside it was a third kind of label on the row.
        <NoFillBadge icon={ticketTypeIcon(type)} color={typeIconColor} title={type} className="mr-0.5 [&>svg]:size-3.5">
          <span className="sr-only">{type}</span>
        </NoFillBadge>
      )}
      {labels.slice(0, CARD_LABELS).map((label) => (
        <LabelChip key={label} label={label} color={labelColors?.[label]} className="gap-1 border-transparent bg-transparent px-0 text-[11px]" />
      ))}
      {labels.length > CARD_LABELS && (
        <span
          title={labels.slice(CARD_LABELS).join(", ")}
          className="font-mono text-[11px] text-muted-foreground"
        >
          +{labels.length - CARD_LABELS}
        </span>
      )}
    </span>
  );
  const run = (
    <>
      {runState === undefined && runQueued && (
        <Clock className="size-3 shrink-0 text-info" role="img" aria-label="Play queued" />
      )}
      {runState === "waiting" && (
        <CircleHelp className="size-3 shrink-0 text-info" role="img" aria-label="Play waiting for an answer" />
      )}
      {runState !== undefined && runState !== "waiting" && (
        <span className="flex items-center gap-1">
          <LoaderCircle className="size-3 shrink-0 animate-spin motion-reduce:animate-none text-warning" role="img" aria-label="Play running" />
          {runStartedAt !== undefined && <RunTimer startedAt={runStartedAt} />}
        </span>
      )}
    </>
  );
  const thread = hasThread && (
    <MessageSquare className="size-3.5 shrink-0 text-muted-foreground" role="img" aria-label="Has a thread" />
  );

  return (
    <>
      {/* The key leads like an eyebrow so the title gets the card's full width; who acts next closes the same line, leaving the pills a row of their own. */}
      <span className="flex h-4 items-center gap-1.5 font-mono text-[11px] text-muted-foreground">
        <span className="tabular-nums">{prefix}-{ticket.number}</span>
        <span className="flex-1" />
        {run}
        {thread}
        {avatar}
      </span>
      <span dir="auto" title={ticket.title} className="line-clamp-3 text-[13px] font-[450] leading-snug break-words">{ticket.title}</span>
      <TicketBlockedLine ticketId={ticket.id} />
      {(type !== "" || labels.length > 0) && pills}
    </>
  );
});

const TicketCardImpl = ({ ticket, collapsed = false }: TicketCardProps) => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const queryClient = useQueryClient();
  // Read at click time, not subscribed: cards re-render on every drag move and only the handlers need the prefix.
  const open = () =>
    navigate(wsPath(ticketPath(ticket, queryClient.getQueryData<Project>([getProjectKey, ticket.project_id])?.prefix)));
  // A card moves by dragging it, so a viewer who can't write the ticket gets a card that only opens.
  const draggable = useCanMoveTickets();
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: ticket.id,
    disabled: { draggable: !draggable, droppable: collapsed },
    data: { type: "card", ticketId: ticket.id, statusId: ticket.status, categoryId: ticket.category_id } satisfies DropTargetData,
    transition: { duration: 200, easing: "ease" },
    animateLayoutChanges: animateNearSlot,
  });
  // The ghost remounts in every column it's dragged through; an entrance there reads as lag.
  const [mountedWhileDragging] = useState(isDragging);

  // dnd-kit's inline `transition` replaces the className's, so append; never opacity, or the drop's handover blinks.
  const dragTransition = transition
    ? `${transition}, border-color 150ms var(--ease-standard), translate 150ms var(--ease-standard)`
    : undefined;

  // Equivalent to dnd-kit's CSS.Transform.toString: translate3d plus per-axis scale while dragging.
  const dragCssTransform = transform
    ? `translate3d(${Math.round(transform.x)}px, ${Math.round(transform.y)}px, 0) scaleX(${transform.scaleX}) scaleY(${transform.scaleY})`
    : undefined;

  return (
    <div
      ref={setNodeRef}
      style={{ transform: dragCssTransform, transition: dragTransition }}
      data-no-enter={mountedWhileDragging || undefined}
      {...(draggable ? listeners : {})}
      {...attributes}
      onClick={open}
      onKeyDown={(event) => {
        // Enter opens the ticket, matching native <button>; every other key goes to dnd-kit, whose pickup is Space.
        if (event.key === "Enter") {
          open();
          return;
        }
        if (draggable) listeners?.onKeyDown?.(event);
      }}
      className={cn(
        "group relative flex select-none flex-col gap-1.5 rounded-lg border border-border bg-card px-2.5 py-2 shadow-card transition-[border-color,translate] duration-150 ease-standard hover:border-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50",
        draggable && "cursor-grab active:cursor-grabbing",
        !draggable && "cursor-pointer",
        // A 1px lift on hover; the before strip keeps the vacated pixel inside the card so the hover never flickers off.
        "before:absolute before:inset-x-0 before:-bottom-px before:h-px motion-safe:hover:-translate-y-px",
        "after:pointer-events-none after:absolute after:inset-0 after:rounded-[inherit] after:opacity-0 after:shadow-elevated after:transition-opacity after:duration-150 after:ease-standard hover:after:opacity-50",
        // TicketCardOverlay carries the "lifted" look; this is just a dimmed placeholder for the slot.
        isDragging && "opacity-40",
      )}
    >
      <span data-wash aria-hidden className="pointer-events-none absolute inset-0 rounded-[inherit] bg-success/15 opacity-0" />
      <TicketCardBody ticket={ticket} />
    </div>
  );
};

// Memoized so a ghost entering one column doesn't re-render every card on the board.
export const TicketCard = memo(TicketCardImpl);
