import { PlusIcon, TagIcon, TicketIcon } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

interface BoardCreateMenuProps {
  onNewTicket: () => void;
  onNewCategory: () => void;
  className?: string;
}

const menuItemClass =
  "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[13.5px] text-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 focus-visible:bg-accent/60";

// One "+" entry point: separate buttons read as unrelated when both just add something to the board.
export const BoardCreateMenu = ({ onNewTicket, onNewCategory, className }: BoardCreateMenuProps) => {
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button size="icon" aria-label="Add" title="Add" className={cn(className)}>
          <PlusIcon />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-48 p-1.5">
        <div className="flex flex-col gap-0.5">
          <button
            type="button"
            className={menuItemClass}
            onClick={() => {
              setOpen(false);
              onNewTicket();
            }}
          >
            <TicketIcon className="size-4 shrink-0" aria-hidden />
            New ticket
          </button>
          <button
            type="button"
            className={menuItemClass}
            onClick={() => {
              setOpen(false);
              onNewCategory();
            }}
          >
            <TagIcon className="size-4 shrink-0" aria-hidden />
            New category
          </button>
        </div>
      </PopoverContent>
    </Popover>
  );
};
