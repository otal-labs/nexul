import { TicketStatus, type Ticket, type TicketStatus as TicketStatusType } from "@/models/Ticket";

interface StatusTransitionButtonsProps {
  ticket: Ticket;
  onTransition: (status: TicketStatusType) => Promise<void> | void;
  disabled?: boolean;
}

const statusLabels: Record<TicketStatusType, string> = {
  [TicketStatus.Open]: "Open",
  [TicketStatus.InProgress]: "In progress",
  [TicketStatus.Done]: "Done",
  [TicketStatus.Closed]: "Closed",
};

const available: Record<TicketStatusType, TicketStatusType[]> = {
  [TicketStatus.Open]: [TicketStatus.InProgress, TicketStatus.Done, TicketStatus.Closed],
  [TicketStatus.InProgress]: [TicketStatus.Done, TicketStatus.Closed],
  [TicketStatus.Done]: [TicketStatus.InProgress, TicketStatus.Closed],
  [TicketStatus.Closed]: [TicketStatus.Open],
};

// Meant to sit inside a Popover anchored to the status badge, not as a standalone row.
export const StatusTransitionButtons = ({
  ticket,
  onTransition,
  disabled = false,
}: StatusTransitionButtonsProps) => {
  const next = available[ticket.status] ?? [];
  return (
    <div className="flex flex-col">
      {next.map((status) => (
        <button
          key={status}
          type="button"
          disabled={disabled}
          onClick={() => void onTransition(status)}
          className="rounded-md px-2 py-1.5 text-left text-xs text-foreground transition-colors duration-150 ease-standard hover:bg-accent disabled:pointer-events-none disabled:opacity-50"
        >
          Move to {statusLabels[status]}
        </button>
      ))}
    </div>
  );
};
