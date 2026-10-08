import { useSortable } from "@dnd-kit/sortable";
import { useQueryClient } from "@tanstack/react-query";
import { CircleHelp, LoaderCircle, MessageSquare } from "lucide-react";
import { memo, useState } from "react";
import { useNavigate } from "react-router";

import { PersonAvatar } from "@/components/PersonAvatar";
import type { DropTargetData } from "@/components/board/dragMove";
import { RunTimer } from "@/components/board/RunTimer";
import { TicketBlockedLine } from "@/components/board/TicketBlockedLine";
import { labelDotColor, pillClass, ticketTypeColor } from "@/components/board/ticketTypeColor";
import { TicketTypeIcon } from "@/components/board/ticketTypeIcon";
import { useFetchChatThreadIndicators } from "@/hooks/ChatHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import { getProjectKey, useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchLabelColors } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import { useTicketRunStartedAt, useTicketRunState } from "@/hooks/TrailHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import { personLabel } from "@/models/Person";
import type { Project } from "@/models/Project";
import { cardPerson, ticketPath, type Ticket } from "@/models/Ticket";

interface TicketCardProps {
  ticket: Ticket;
}

interface TicketCardBodyProps {
  ticket: Ticket;
}

// Shared with TicketCardOverlay; memoized because dnd-kit re-renders every sortable on each pointer move.
export const TicketCardBody = memo(({ ticket }: TicketCardBodyProps) => {
  // All five queries are cached/deduped across every mounted TicketCard, so a column of 50 fires one request each.
  const { data: project } = useFetchProject(ticket.project_id);
  const { data: ticketTypes } = useFetchProjectTicketTypes(ticket.project_id);
  const { data: labelColors } = useFetchLabelColors(ticket.project_id);
  const { data: threadIndicators } = useFetchChatThreadIndicators(ticket.project_id);
  const { data: statuses } = useFetchProjectStatuses(ticket.project_id);
  const person = cardPerson(ticket, statuses?.find((s) => s.id === ticket.status)?.kind);
  const shown = usePerson(person.login);
  const hasThread = threadIndicators?.[ticket.id] === true;
  const runState = useTicketRunState(ticket.project_id, ticket.id);
  const runStartedAt = useTicketRunStartedAt(ticket.project_id, ticket.id);
  const prefix = project?.prefix ?? "";
  const ticketType = ticketTypes?.find((t) => t.id === ticket.type_id);
  const type = ticketType?.name ?? "";
  const labels = ticket.labels ?? [];

  return (
    <>
      {/* The whole card is the drag handle and click target; a still click never activates dnd-kit, so no inner handler is needed. */}
      <span className="flex items-center gap-2.5">
        {person.login && (
          <span role="img" aria-label={`${person.role} ${personLabel(shown)}`} className="shrink-0">
            <PersonAvatar login={person.login} src={shown.avatar_url} className="size-7 text-xs" />
          </span>
        )}
        <span title={ticket.title} className="line-clamp-3 min-w-0 flex-1 text-sm font-medium leading-snug break-words">
          {ticket.title}
        </span>
        {hasThread && (
          <MessageSquare className="size-3.5 shrink-0 text-muted-foreground" role="img" aria-label="Has a chat thread" />
        )}
      </span>
      <TicketBlockedLine ticketId={ticket.id} />
      {/* Tinted pills for type and labels, ticket id on the right; color stays inside the pills, never on the card. */}
      <span className="flex items-end justify-between gap-2">
        <span className="flex min-w-0 flex-wrap items-center gap-1.5">
          {type !== "" && (
            <span
              data-slot="pill"
              className={cn("inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium", pillClass(ticketTypeColor(type, ticketType?.color)))}
            >
              <TicketTypeIcon typeName={type} className="size-3" aria-hidden />
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
        <span className="flex shrink-0 items-center gap-1.5 font-mono text-xs text-muted-foreground">
          {runState === "waiting" && (
            <CircleHelp className="size-3 shrink-0 text-info" role="img" aria-label="A play is waiting for an answer" />
          )}
          {runState !== undefined && runState !== "waiting" && (
            <span className="flex items-center gap-1">
              <LoaderCircle className="size-3 shrink-0 animate-spin motion-reduce:animate-none text-warning" role="img" aria-label="A play is running" />
              {runStartedAt !== undefined && <RunTimer startedAt={runStartedAt} />}
            </span>
          )}
          {prefix}-{ticket.number}
        </span>
      </span>
    </>
  );
});

const TicketCardImpl = ({ ticket }: TicketCardProps) => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const queryClient = useQueryClient();
  // Read at click time, not subscribed: cards re-render on every drag move and only the handlers need the prefix.
  const open = () =>
    navigate(wsPath(ticketPath(ticket, queryClient.getQueryData<Project>([getProjectKey, ticket.project_id])?.prefix)));
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: ticket.id,
    data: { type: "card", ticketId: ticket.id, statusId: ticket.status, categoryId: ticket.category_id } satisfies DropTargetData,
    transition: { duration: 200, easing: "ease" },
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
      {...listeners}
      {...attributes}
      onClick={open}
      onKeyDown={(event) => {
        // Space is dnd-kit's keyboard drag pickup; only Enter opens the ticket, matching native <button>.
        if (event.key === "Enter") open();
      }}
      className={cn(
        "group relative flex select-none flex-col gap-2.5 rounded-lg border border-border bg-card p-3 transition-[border-color,translate] duration-150 ease-standard hover:border-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50",
        "cursor-grab active:cursor-grabbing",
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
