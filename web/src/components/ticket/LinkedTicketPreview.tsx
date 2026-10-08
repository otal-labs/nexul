import { useMemo } from "react";

import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTicket } from "@/hooks/TicketHooks";
import { linkedTicketKey, type LinkedTicket } from "@/models/TicketLink";
import { bodyToPlainText } from "@/utils/RichtextUtility";

interface LinkedTicketPreviewProps {
  ticket: LinkedTicket;
}

// Mounts only while the hover card is open, so the full ticket is fetched on first hover, not per row.
export const LinkedTicketPreview = ({ ticket }: LinkedTicketPreviewProps) => {
  const { data: full } = useFetchTicket(ticket.id);
  const { data: statuses } = useFetchProjectStatuses(ticket.project_id);
  const statusName = statuses?.find((s) => s.id === ticket.status)?.name;
  const description = useMemo(() => (full ? bodyToPlainText(full.body) : undefined), [full]);

  return (
    <div className="space-y-1.5">
      <p className="font-mono text-[11px] text-muted-foreground">
        {linkedTicketKey(ticket)}
        {statusName && ` · ${statusName}`}
      </p>
      <p className="text-sm font-medium break-words">{ticket.title}</p>
      {description && <p className="line-clamp-2 text-xs text-muted-foreground">{description}</p>}
      {description === "" && <p className="text-xs text-muted-foreground">No description.</p>}
    </div>
  );
};
