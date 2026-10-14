import { Ban, CheckCircle2, Circle } from "lucide-react";
import { Link } from "react-router";

import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { LinkedTicketPreview } from "@/components/ticket/LinkedTicketPreview";
import { RemoveLinkButton } from "@/components/ticket/RemoveLinkButton";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import { linkedTicketKey, type LinkedTicket } from "@/models/TicketLink";

interface LinkedTicketRowProps {
  ticket: LinkedTicket;
  /** A blocker the ticket still waits on shows the blocked icon instead of a neutral circle. */
  waiting?: boolean;
  onRemove?: () => void;
  /** The project of the ticket this row hangs off, whose links the remove writes. */
  hostProjectId?: string;
}

const stateOf = (ticket: LinkedTicket, waiting: boolean) => {
  if (ticket.done) return { Icon: CheckCircle2, color: "text-success", label: "Done" };
  if (waiting) return { Icon: Ban, color: "text-warning", label: "Not done yet" };
  return { Icon: Circle, color: "text-muted-foreground", label: "Not done" };
};

export const LinkedTicketRow = ({ ticket, waiting = false, onRemove, hostProjectId }: LinkedTicketRowProps) => {
  const { Icon, color, label } = stateOf(ticket, waiting);
  const key = linkedTicketKey(ticket);
  const wsPath = useWorkspacePath();

  return (
    <li className="group flex items-center gap-2 rounded-md px-2 py-0.5 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <Icon className={cn("size-3.5 shrink-0", color)} role="img" aria-label={label} />
      <HoverCard openDelay={300} closeDelay={100}>
        <HoverCardTrigger asChild>
          <Link
            to={wsPath(`/tickets/${key}`)}
            aria-label={`${key} ${ticket.title}`}
            className="mr-auto min-w-0 truncate py-1 font-mono text-xs underline-offset-4 hover:underline"
          >
            {key}
          </Link>
        </HoverCardTrigger>
        <HoverCardContent side="left" align="start" sideOffset={32} className="w-72 p-3">
          <LinkedTicketPreview ticket={ticket} />
        </HoverCardContent>
      </HoverCard>
      {onRemove && (
        <RemoveLinkButton projectId={hostProjectId} label={`Remove link to ${key}`} onRemove={onRemove} />
      )}
    </li>
  );
};
