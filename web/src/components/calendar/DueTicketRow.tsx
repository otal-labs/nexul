import { CalendarClockIcon } from "lucide-react";
import { Link } from "react-router";

import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import { ticketPath, type Ticket } from "@/models/Ticket";

interface DueTicketRowProps {
  ticket: Ticket;
  detailed?: boolean;
}

// A ticket on its due day: its key and title, linking to the ticket.
export const DueTicketRow = ({ ticket, detailed = false }: DueTicketRowProps) => {
  const wsPath = useWorkspacePath();
  const { data: projects } = useFetchProjects();
  const prefix = projects?.find((p) => p.id === ticket.project_id)?.prefix;
  const key = prefix ? `${prefix}-${ticket.number}` : "Ticket";
  return (
    <Link
      to={wsPath(ticketPath(ticket, prefix))}
      className={cn(
        "flex w-full min-w-0 flex-wrap items-center gap-x-2 rounded-md px-1.5 py-0.5 text-xs transition-colors duration-150 ease-standard hover:bg-accent/40",
        detailed && "py-1.5 text-sm",
      )}
    >
      <CalendarClockIcon className="size-3 shrink-0 text-muted-foreground" aria-hidden />
      <span className="sr-only">Due: </span>
      <span className="shrink-0 font-mono text-muted-foreground">{key}</span>
      <span className="min-w-0 flex-1 basis-20 truncate">{ticket.title}</span>
    </Link>
  );
};
