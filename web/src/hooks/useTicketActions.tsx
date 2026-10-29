import { useNavigate } from "react-router";

import type { RowAction } from "@/components/settings/RowActionsMenu";
import { useDeleteTicket } from "@/hooks/TicketHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useIsEmbedded } from "@/hooks/useEmbeddedCrumbs";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Project } from "@/models/Project";
import { boardPath } from "@/models/Project";
import type { Ticket } from "@/models/Ticket";

// After Delete, a ticket shown inside another page (the Inbox) stays put; a ticket page returns to its board.
export const useTicketActions = (ticket: Ticket, label: string, project: Project | undefined, onHistory: () => void): RowAction[] => {
  const embedded = useIsEmbedded();
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { open: confirm } = useConfirmationDialog();
  const deleteTicket = useDeleteTicket();

  const remove = async () => {
    const ok = await confirm({
      title: `Delete ${label}?`,
      message: `"${ticket.title}" is deleted for good, with its labels, links, attachments and thread. Tickets it blocks are no longer blocked. This can't be undone.`,
      confirmLabel: "Delete ticket",
    });
    if (!ok) return;
    deleteTicket.mutate(ticket.id, {
      onSuccess: () => {
        if (!embedded && project) void navigate(wsPath(boardPath(project)));
      },
    });
  };

  return [
    { label: "History", permission: "tickets:read", verb: "read this ticket's history", onSelect: onHistory },
    { label: "Delete", destructive: true, permission: "tickets:delete", onSelect: () => void remove() },
  ];
};
