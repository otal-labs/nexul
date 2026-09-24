import { Ban, Bug, PlusIcon } from "lucide-react";
import { useState } from "react";

import { TicketPickerList } from "@/components/ticket/TicketPickerList";
import { menuItemClass } from "@/components/ticket/ticketFormPillStyles";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useAddBlocker, useSetFoundIn } from "@/hooks/TicketLinkHooks";

type PickKind = "blocked_by" | "found_in";

interface AddTicketLinkMenuProps {
  ticketId: string;
}

// "+" menu: pick the link kind, then the ticket; a new found-in replaces the old one.
export const AddTicketLinkMenu = ({ ticketId }: AddTicketLinkMenuProps) => {
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<PickKind | null>(null);
  const addBlocker = useAddBlocker();
  const setFoundIn = useSetFoundIn();

  const close = (next: boolean) => {
    setOpen(next);
    if (!next) setKind(null);
  };

  const pick = (otherId: string) => {
    if (kind === "blocked_by") addBlocker.mutate({ id: ticketId, blockerId: otherId });
    if (kind === "found_in") setFoundIn.mutate({ id: ticketId, originId: otherId });
    close(false);
  };

  return (
    <Popover open={open} onOpenChange={close}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="Link a ticket"
          className="flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-muted/50 hover:text-foreground"
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
            <button type="button" className={menuItemClass} onClick={() => setKind("found_in")}>
              <Bug className="size-3.5" aria-hidden /> Found in…
            </button>
          </div>
        )}
        {kind !== null && <TicketPickerList excludeId={ticketId} onSelect={pick} />}
      </PopoverContent>
    </Popover>
  );
};
