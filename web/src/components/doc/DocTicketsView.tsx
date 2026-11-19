import { Link } from "react-router";

import { DocTicketRow } from "@/components/doc/DocTicketRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useDoneCount } from "@/hooks/StatusHooks";
import { useFetchTicketsByDoc } from "@/hooks/TicketHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { boardPath } from "@/models/Project";

interface DocTicketsViewProps {
  docId: string;
  projectId: string;
}

// The doc page's Tickets view: the count line, then one row per ticket in the order they were written.
export const DocTicketsView = ({ docId, projectId }: DocTicketsViewProps) => {
  const wsPath = useWorkspacePath();
  const { data: tickets, error, isPending } = useFetchTicketsByDoc(docId);
  const { data: project } = useFetchProject(projectId);
  const done = useDoneCount(tickets ?? []);
  return (
    <section aria-label="Tickets">
      {isPending && <LoadingDisplay label="Loading tickets…" />}
      {error && <ErrorDisplay error={error} />}
      {tickets && (
        <p className="flex flex-wrap items-center gap-x-1.5 text-sm text-muted-foreground">
          <span className="font-mono tabular-nums text-foreground">{tickets.length}</span>
          <span>{tickets.length === 1 ? "ticket" : "tickets"}</span>
          <span aria-hidden>·</span>
          <span className="font-mono tabular-nums">{done}</span>
          <span>done</span>
          {project && <span aria-hidden>·</span>}
          {project && (
            <Link
              to={wsPath(boardPath(project))}
              className="underline-offset-4 transition-colors duration-150 ease-standard hover:text-foreground hover:underline"
            >
              Open board
            </Link>
          )}
        </p>
      )}
      {tickets && tickets.length > 0 && (
        <ul className="-mx-2 mt-3 grid grid-cols-[0.875rem_auto_minmax(0,1fr)] gap-x-3">
          {tickets.map((ticket) => (
            <DocTicketRow key={ticket.id} ticket={ticket} />
          ))}
        </ul>
      )}
    </section>
  );
};
