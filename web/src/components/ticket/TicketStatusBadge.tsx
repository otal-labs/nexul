import { StatusMark } from "@/components/board/StatusIcon";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { StatusKind } from "@/models/Status";
import { TicketStatus, type Ticket } from "@/models/Ticket";

// Tickets filed before statuses became per-project columns can still hold one of these values.
const legacyStatuses: Record<string, { name: string; kind: StatusKind; icon: string }> = {
  [TicketStatus.Open]: { name: "Open", kind: StatusKind.Backlog, icon: "" },
  [TicketStatus.InProgress]: { name: "In progress", kind: StatusKind.Progress, icon: "" },
  [TicketStatus.Done]: { name: "Done", kind: StatusKind.Done, icon: "" },
  [TicketStatus.Closed]: { name: "Closed", kind: StatusKind.Backlog, icon: "CircleX" },
};

interface TicketStatusBadgeProps {
  ticket: Pick<Ticket, "project_id" | "status">;
}

export const TicketStatusBadge = ({ ticket }: TicketStatusBadgeProps) => {
  const { data: statuses, isPending } = useFetchProjectStatuses(ticket.project_id);
  const status = statuses?.find((s) => s.id === ticket.status) ?? legacyStatuses[ticket.status];
  if (!status && isPending) return null;
  return (
    <span className="inline-flex w-fit items-center gap-1.5 text-xs font-medium">
      {status && <StatusMark status={status} className="size-3" />}
      {status && status.name}
      {!status && <span className="text-muted-foreground">Unknown status</span>}
    </span>
  );
};
