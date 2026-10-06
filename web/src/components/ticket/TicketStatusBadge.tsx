import { StatusMark } from "@/components/board/StatusIcon";
import { useTicketStatus } from "@/hooks/StatusHooks";
import type { Ticket } from "@/models/Ticket";

interface TicketStatusBadgeProps {
  ticket: Pick<Ticket, "project_id" | "status">;
}

export const TicketStatusBadge = ({ ticket }: TicketStatusBadgeProps) => {
  const { status, isPending } = useTicketStatus(ticket);
  if (!status && isPending) return null;
  return (
    <span className="inline-flex w-fit items-center gap-1.5 text-xs font-medium">
      {status && <StatusMark status={status} className="size-3" />}
      {status && status.name}
      {!status && <span className="text-muted-foreground">Unknown status</span>}
    </span>
  );
};
