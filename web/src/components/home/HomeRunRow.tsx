import { Link } from "react-router";

import { formatShortAgo } from "@/components/doc/docTime";
import type { TicketRun } from "@/components/home/homeRuns";
import { ticketQuery } from "@/hooks/TicketHooks";
import { useHoverPrefetch } from "@/hooks/useHoverPrefetch";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import { ticketPath } from "@/models/Ticket";

interface HomeRunRowProps {
  run: TicketRun;
  prefix: string;
}

export const HomeRunRow = ({ run: { ticket, waiting, started }, prefix }: HomeRunRowProps) => {
  const wsPath = useWorkspacePath();
  const prefetch = useHoverPrefetch((client) => client.prefetchQuery(ticketQuery(ticket.id)));
  return (
    <li>
      <Link
        to={wsPath(ticketPath(ticket, prefix))}
        {...prefetch}
        className="-mx-2 flex gap-2 rounded-md px-2 py-2 text-row transition-colors duration-150 ease-standard hover:bg-accent inset-focus"
      >
        <span className="flex h-lh w-3.5 shrink-0 items-center justify-center">
          <span
            aria-hidden
            className={cn("size-2 rounded-full", waiting ? "bg-warning-mark" : "animate-[status-pulse_2.4s_ease-standard_infinite] bg-info")}
          />
        </span>
        <span className="min-w-0 flex-1">
          <span className="flex items-baseline gap-1.5">
            <span dir="auto" title={ticket.title} className="truncate font-medium text-foreground">
              {ticket.title}
            </span>
            {started !== "" && (
              <span className="ml-auto shrink-0 pl-3 font-mono text-micro text-muted-foreground tabular-nums">{formatShortAgo(started)}</span>
            )}
          </span>
          <span className="block truncate text-xs text-muted-foreground">
            <span className="font-mono">{prefix ? `${prefix}-${ticket.number}` : ticket.number}</span>
            {" · "}
            {waiting ? "waiting for an answer" : "running"}
          </span>
        </span>
      </Link>
    </li>
  );
};
