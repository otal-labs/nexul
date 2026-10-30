import { RowActions } from "@/components/listpane/RowActions";
import { useFetchOrCreateTicketThread, useFetchTicketThreadStatus } from "@/hooks/ChatHooks";
import { useBotsDialog } from "@/hooks/useBotsDialog";

interface TicketThreadMenuProps {
  workspaceId: string;
  ticketId: string;
}

// The ⋯ of a ticket's thread, drawn by whichever row names the thread (the page's tab row, the section's heading) so it never floats alone.
export const TicketThreadMenu = ({ workspaceId, ticketId }: TicketThreadMenuProps) => {
  const { data: status } = useFetchTicketThreadStatus(ticketId);
  const { data: conversation } = useFetchOrCreateTicketThread(workspaceId, ticketId, status?.[ticketId] === true);
  const { onBots, botsDialog } = useBotsDialog(conversation, "Ticket thread");

  return (
    <>
      {onBots && <RowActions itemLabel="the thread" onBots={onBots} />}
      {botsDialog}
    </>
  );
};
