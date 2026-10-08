import { Bug, PlusIcon, TagIcon, TicketIcon } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { menuItemClass } from "@/components/MenuItem";
import { useReportBugDialog } from "@/hooks/useReportBugDialog";

interface BoardCreateMenuProps {
  onNewTicket: () => void;
  onNewCategory: () => void;
}

// One "+" entry point: separate buttons read as unrelated when both just add something to the board.
export const BoardCreateMenu = ({ onNewTicket, onNewCategory }: BoardCreateMenuProps) => {
  const [open, setOpen] = useState(false);
  const reportBug = useReportBugDialog();

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button size="sm" className="h-9 px-3">
          <PlusIcon />
          Create
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-48 p-1">
        <div className="flex flex-col">
          <button
            type="button"
            className={menuItemClass}
            onClick={() => {
              setOpen(false);
              onNewTicket();
            }}
          >
            <TicketIcon aria-hidden />
            New ticket
          </button>
          <button
            type="button"
            className={menuItemClass}
            onClick={() => {
              setOpen(false);
              void reportBug();
            }}
          >
            <Bug aria-hidden />
            Report a bug
          </button>
          <button
            type="button"
            className={menuItemClass}
            onClick={() => {
              setOpen(false);
              onNewCategory();
            }}
          >
            <TagIcon aria-hidden />
            New category
          </button>
        </div>
      </PopoverContent>
    </Popover>
  );
};
