import { Link } from "react-router";

import { StatusMark } from "@/components/board/StatusIcon";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useTicketStatus } from "@/hooks/StatusHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { ticketPath, type Ticket } from "@/models/Ticket";

interface DocTicketRowProps {
  ticket: Ticket;
}

// One 32px row of a doc's Tickets view: the column's mark, the mono key and the title, the whole row a link.
export const DocTicketRow = ({ ticket }: DocTicketRowProps) => {
  const wsPath = useWorkspacePath();
  const { data: project } = useFetchProject(ticket.project_id);
  const { status } = useTicketStatus(ticket);
  const key = project ? `${project.prefix}-${ticket.number}` : "";
  return (
    <li className="col-span-3 grid grid-cols-subgrid">
      <Link
        to={wsPath(ticketPath(ticket, project?.prefix))}
        title={status ? `${status.name}: ${ticket.title}` : ticket.title}
        className="col-span-3 grid h-8 grid-cols-subgrid items-center rounded-sm px-2 text-row transition-colors duration-150 ease-standard hover:bg-accent"
      >
        <span className="flex size-3.5 items-center justify-center">{status && <StatusMark status={status} className="size-3.5" />}</span>
        <span className="font-mono text-xs text-muted-foreground tabular-nums">{key}</span>
        <span dir="auto" className="min-w-0 truncate">{ticket.title}</span>
      </Link>
    </li>
  );
};
