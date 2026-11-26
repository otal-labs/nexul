import { Link } from "react-router";

import { StatusMark } from "@/components/board/StatusIcon";
import { formatShortAgo } from "@/components/doc/docTime";
import type { TicketOnYou } from "@/components/home/homeTickets";
import { ticketQuery } from "@/hooks/TicketHooks";
import { useHoverPrefetch } from "@/hooks/useHoverPrefetch";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { StatusKind } from "@/models/Status";
import { ticketPath } from "@/models/Ticket";
import { formatFullTime } from "@/utils/TimeUtility";

// A status without an icon of its own takes its stage's, so the mark says "not started" or "in review" instead of a bare dot.
const stageIcon: Record<StatusKind, string> = {
  [StatusKind.Backlog]: "CircleDashed",
  [StatusKind.Progress]: "CircleDot",
  [StatusKind.Review]: "CircleEllipsis",
  [StatusKind.Testing]: "CircleEllipsis",
  [StatusKind.Done]: "CircleCheckBig",
};

interface HomeTicketRowProps {
  item: TicketOnYou;
  prefix: string;
}

export const HomeTicketRow = ({ item: { ticket, status }, prefix }: HomeTicketRowProps) => {
  const wsPath = useWorkspacePath();
  const prefetch = useHoverPrefetch((client) => client.prefetchQuery(ticketQuery(ticket.id)));
  return (
    <li>
      <Link
        to={wsPath(ticketPath(ticket, prefix))}
        {...prefetch}
        className="-mx-2 flex gap-2.5 rounded-md px-2 py-2 text-row transition-colors duration-150 ease-standard hover:bg-accent inset-focus"
      >
        <span className="flex h-lh w-4 shrink-0 items-center justify-center">
          {status && <StatusMark status={{ kind: status.kind, icon: status.icon || stageIcon[status.kind] }} className="size-3.5" />}
        </span>
        <span className="min-w-0 flex-1">
          <span className="flex items-baseline gap-1.5">
            <span dir="auto" title={ticket.title} className="truncate font-medium text-foreground">
              {ticket.title}
            </span>
            <span title={`Updated ${formatFullTime(ticket.updated_at)}`} className="ml-auto shrink-0 pl-3 font-mono text-micro text-muted-foreground tabular-nums">
              {formatShortAgo(ticket.updated_at)}
            </span>
          </span>
          <span className="block truncate text-xs text-muted-foreground">
            <span className="font-mono">{prefix ? `${prefix}-${ticket.number}` : ticket.number}</span>
            {" · "}
            {status?.name ?? "Unknown status"}
            {status?.kind === StatusKind.Testing && " · yours to test"}
          </span>
        </span>
      </Link>
    </li>
  );
};
