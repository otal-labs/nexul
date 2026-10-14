import { useNavigate } from "react-router";

import type { RowMenu } from "@/components/listpane/rowMenu";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useDeleteTicket } from "@/hooks/TicketHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useIsEmbedded } from "@/hooks/useEmbeddedCrumbs";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Project } from "@/models/Project";
import { boardPath } from "@/models/Project";
import type { Ticket } from "@/models/Ticket";

// A ticket's ⋯ menu: Delete is left out when the viewer lacks it, and the server still checks. A ticket shown inside
// another page (the Inbox) stays where it is; a ticket page returns to its project's board.
export const useTicketActions = (ticket: Ticket, label: string, project: Project | undefined): RowMenu => {
  const canDelete = useAreaAccess(ticket.project_id)?.("deleteTickets") ?? false;
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

  return { onDelete: canDelete ? () => void remove() : undefined };
};
