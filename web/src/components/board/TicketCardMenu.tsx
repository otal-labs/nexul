import { LinkIcon, SquareArrowOutUpRightIcon, Trash2Icon } from "lucide-react";
import type { ReactElement } from "react";
import { toast } from "sonner";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { TicketCardPropertyMenus } from "@/components/board/TicketCardPropertyMenus";
import { contextParts } from "@/components/listpane/rowMenu";
import { ContextMenu, ContextMenuContent, ContextMenuItem, ContextMenuTrigger } from "@/components/ui/context-menu";
import { useBoardCardData } from "@/hooks/useBoardCardData";
import { useConfirmDeleteTicket } from "@/hooks/useTicketActions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { ticketPath, type Ticket } from "@/models/Ticket";

interface TicketCardMenuProps {
  ticket: Ticket;
  /** The card itself: it takes the trigger's handlers and its data-state. */
  children: ReactElement;
  /** Off while the card is being dragged. */
  disabled: boolean;
  onOpen: () => void;
}

interface TicketCardMenuItemsProps {
  ticket: Ticket;
  onOpen: () => void;
}

// Mounted only while the menu is open, so a board of cards holds no pickers until one is asked for.
const TicketCardMenuItems = ({ ticket, onOpen }: TicketCardMenuItemsProps) => {
  const { prefix } = useBoardCardData();
  const wsPath = useWorkspacePath();
  const confirmDelete = useConfirmDeleteTicket();
  const key = prefix ? `${prefix}-${ticket.number}` : ticket.title;

  const copyLink = async () => {
    try {
      await navigator.clipboard.writeText(new URL(wsPath(ticketPath(ticket, prefix)), window.location.origin).href);
      toast.success("Link copied");
    } catch {
      toast.error("The browser refused the clipboard");
    }
  };

  return (
    <>
      <ContextMenuItem onSelect={onOpen}>
        <SquareArrowOutUpRightIcon aria-hidden />
        Open ticket
      </ContextMenuItem>
      <ContextMenuItem onSelect={() => void copyLink()}>
        <LinkIcon aria-hidden />
        Copy link
      </ContextMenuItem>
      <TicketCardPropertyMenus ticket={ticket} />
      <ActionMenuItem
        permission="tickets:delete"
        projectId={ticket.project_id}
        ui={contextParts}
        icon={Trash2Icon}
        destructive
        separated
        onSelect={() => void confirmDelete(ticket, key)}
      >
        Delete
      </ActionMenuItem>
    </>
  );
};

// A right-click on a card edits its properties in place, the way the ticket rail does, without leaving the board.
export const TicketCardMenu = ({ ticket, children, disabled, onOpen }: TicketCardMenuProps) => (
  <ContextMenu>
    <ContextMenuTrigger asChild disabled={disabled}>
      {children}
    </ContextMenuTrigger>
    <ContextMenuContent className="w-60">
      <TicketCardMenuItems ticket={ticket} onOpen={onOpen} />
    </ContextMenuContent>
  </ContextMenu>
);
