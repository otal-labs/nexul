import { XIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { useFetchTicket } from "@/hooks/TicketHooks";

interface UnlinkButtonProps {
  ticketId: string;
  /** What it unlinks, for its name: "branch ticket/1". */
  subject: string;
  onUnlink: () => void;
}

// Shown on hover and focus; locked or hidden for someone who may not edit the ticket's links.
export const UnlinkButton = ({ ticketId, subject, onUnlink }: UnlinkButtonProps) => {
  const { data: ticket } = useFetchTicket(ticketId);
  return (
    <ActionButton
      permission="tickets:write"
      projectId={ticket?.project_id}
      verb="unlink development items"
      variant="ghost"
      size="icon"
      icon={XIcon}
      aria-label={`Unlink ${subject}`}
      title={`Unlink ${subject}`}
      onClick={onUnlink}
      className="size-5 shrink-0 text-muted-foreground opacity-0 transition-[color,background-color,opacity] duration-150 ease-standard group-focus-within:opacity-100 group-hover:opacity-100 [&_svg]:size-3"
    />
  );
};
