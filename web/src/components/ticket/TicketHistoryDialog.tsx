import { HistoryDialog } from "@/components/history/HistoryDialog";
import { ticketHistoryUrl, ticketVersionUrl, useRestoreTicketVersion } from "@/hooks/TicketHistoryHooks";
import type { Ticket } from "@/models/Ticket";

interface TicketHistoryDialogProps {
  ticket: Ticket;
  onClose: () => void;
}

export const TicketHistoryDialog = ({ ticket, onClose }: TicketHistoryDialogProps) => {
  const restore = useRestoreTicketVersion(ticket.id);

  return (
    <HistoryDialog
      historyUrl={ticketHistoryUrl(ticket.id)}
      versionUrl={(version) => ticketVersionUrl(ticket.id, version)}
      name={ticket.title}
      retention="ten"
      restore={{
        permission: "tickets:write",
        projectId: ticket.project_id,
        run: (version) => restore.mutate(version, { onSuccess: onClose }),
        pending: restore.isPending,
      }}
      onClose={onClose}
    />
  );
};
