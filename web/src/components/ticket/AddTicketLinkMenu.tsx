import { Ban, Bug, FileText, OctagonX, PlusIcon } from "lucide-react";
import { useState } from "react";

import { DocPickerList } from "@/components/doc/DocPickerList";
import { TicketPickerList } from "@/components/ticket/TicketPickerList";
import { menuItemClass } from "@/components/ticket/ticketFormPillStyles";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useSetTicketSource } from "@/hooks/TicketHooks";
import { useAddBlocker, useSetFoundIn } from "@/hooks/TicketLinkHooks";
import type { Ticket } from "@/models/Ticket";

type PickKind = "blocked_by" | "blocks" | "found_in" | "source";

interface AddTicketLinkMenuProps {
  ticket: Ticket;
}

// "+" menu: pick the link kind, then the ticket or doc; a new found-in or source replaces the old one.
// Blocks is the same link held by the other ticket.
export const AddTicketLinkMenu = ({ ticket }: AddTicketLinkMenuProps) => {
  const ticketId = ticket.id;
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<PickKind | null>(null);
  const addBlocker = useAddBlocker();
  const setFoundIn = useSetFoundIn();
  const setSource = useSetTicketSource();

  const close = (next: boolean) => {
    setOpen(next);
    if (!next) setKind(null);
  };

  const pick = (otherId: string) => {
    if (kind === "blocked_by") addBlocker.mutate({ id: ticketId, blockerId: otherId });
    if (kind === "blocks") addBlocker.mutate({ id: otherId, blockerId: ticketId });
    if (kind === "found_in") setFoundIn.mutate({ id: ticketId, originId: otherId });
    if (kind === "source") setSource.mutate({ id: ticketId, docId: otherId });
    close(false);
  };

  return (
    <Popover open={open} onOpenChange={close}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="Add a link"
          className="flex size-5 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-muted/50 hover:text-foreground"
        >
          <PlusIcon className="size-3.5" aria-hidden />
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-72 max-w-[calc(100vw-2rem)] p-1.5">
        {kind === null && (
          <div className="flex flex-col gap-0.5">
            <button type="button" className={menuItemClass} onClick={() => setKind("blocked_by")}>
              <Ban className="size-3.5" aria-hidden /> Blocked by…
            </button>
            <button type="button" className={menuItemClass} onClick={() => setKind("blocks")}>
              <OctagonX className="size-3.5" aria-hidden /> Blocks…
            </button>
            <button type="button" className={menuItemClass} onClick={() => setKind("found_in")}>
              <Bug className="size-3.5" aria-hidden /> Found in…
            </button>
            <button type="button" className={menuItemClass} onClick={() => setKind("source")}>
              <FileText className="size-3.5" aria-hidden /> Source doc…
            </button>
          </div>
        )}
        {(kind === "blocked_by" || kind === "blocks" || kind === "found_in") && <TicketPickerList excludeId={ticketId} onSelect={pick} />}
        {kind === "source" && <DocPickerList projectId={ticket.project_id} excludeId={ticket.doc_id} onSelect={pick} />}
      </PopoverContent>
    </Popover>
  );
};
