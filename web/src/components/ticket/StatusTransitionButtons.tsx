import { CheckIcon } from "lucide-react";

import { StatusMark } from "@/components/board/StatusIcon";
import { menuItemClass } from "@/components/ticket/ticketPropertyRowStyle";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

interface StatusTransitionButtonsProps {
  ticket: Ticket;
  onTransition: (statusId: string) => Promise<void> | void;
}

// Meant to sit inside a Popover anchored to the status badge, not as a standalone row.
export const StatusTransitionButtons = ({ ticket, onTransition }: StatusTransitionButtonsProps) => {
  const { data: statuses } = useFetchProjectStatuses(ticket.project_id);
  return (
    <div className="flex flex-col gap-0.5">
      {statuses?.map((status) => (
        <StatusOption
          key={status.id}
          status={status}
          current={status.id === ticket.status}
          onPick={() => void onTransition(status.id)}
        />
      ))}
    </div>
  );
};

interface StatusOptionProps {
  status: BoardStatus;
  current: boolean;
  onPick: () => void;
}

const StatusOption = ({ status, current, onPick }: StatusOptionProps) => (
  <button type="button" className={menuItemClass} onClick={current ? undefined : onPick}>
    <StatusMark status={status} className="size-3.5" />
    <span className="min-w-0 flex-1 truncate">{status.name}</span>
    {current && <CheckIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />}
  </button>
);
