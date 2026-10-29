import { CalendarClockIcon } from "lucide-react";
import { Link } from "react-router";

import { DUE_BOX, ENTRY_ROW, WEEK_ROW } from "@/components/calendar/entryLayout";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import { ticketPath, type Ticket } from "@/models/Ticket";

interface DueTicketRowProps {
  ticket: Ticket;
  detailed?: boolean;
}

// A ticket on its due day: in the month a box holding its title, in the week a row with the key after the title.
export const DueTicketRow = ({ ticket, detailed = false }: DueTicketRowProps) => {
  const wsPath = useWorkspacePath();
  const { data: projects } = useFetchProjects();
  const prefix = projects?.find((p) => p.id === ticket.project_id)?.prefix;
  const key = prefix ? `${prefix}-${ticket.number}` : "Ticket";
  return (
    <Link
      to={wsPath(ticketPath(ticket, prefix))}
      aria-label={`Due: ${key} ${ticket.title}`}
      title={`${key} ${ticket.title}`}
      className={detailed ? cn(ENTRY_ROW, WEEK_ROW, "text-sm") : DUE_BOX}
    >
      <span className={cn("flex items-center gap-1.5 text-muted-foreground", !detailed && "@max-[55rem]:hidden")}>
        {detailed && <span aria-hidden className="size-1.5 shrink-0" />}
        <CalendarClockIcon className="size-3 shrink-0" aria-hidden />
        {detailed && <span className="relative top-px text-xs">Due</span>}
      </span>
      <span className="flex min-w-0 items-baseline gap-2">
        <span className="truncate">{ticket.title}</span>
        {detailed && <span className="shrink-0 font-mono text-xs text-muted-foreground">{key}</span>}
      </span>
    </Link>
  );
};
