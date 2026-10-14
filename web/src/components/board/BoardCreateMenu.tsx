import { Bug, PlusIcon, TagIcon, TicketIcon } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ActionMenuItem } from "@/components/action/ActionMenuItem";
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
          <ActionMenuItem
            permission="tickets:write"
            icon={TicketIcon}
            onSelect={() => {
              setOpen(false);
              onNewTicket();
            }}
          >
            New ticket
          </ActionMenuItem>
          <ActionMenuItem
            permission="tickets:write"
            icon={Bug}
            onSelect={() => {
              setOpen(false);
              void reportBug();
            }}
          >
            Report a bug
          </ActionMenuItem>
          <ActionMenuItem
            permission="projects:write"
            icon={TagIcon}
            onSelect={() => {
              setOpen(false);
              onNewCategory();
            }}
          >
            New category
          </ActionMenuItem>
        </div>
      </PopoverContent>
    </Popover>
  );
};
