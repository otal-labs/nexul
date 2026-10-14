import { Bug, PlusIcon, TagIcon, TicketIcon } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { useActionStates } from "@/hooks/useActionState";
import { useReportBugDialog } from "@/hooks/useReportBugDialog";

interface BoardCreateMenuProps {
  projectId?: string | undefined;
  onNewTicket: () => void;
  onNewCategory: () => void;
}

// One "+" entry point: separate buttons read as unrelated when both just add something to the board.
export const BoardCreateMenu = ({ projectId, onNewTicket, onNewCategory }: BoardCreateMenuProps) => {
  const [open, setOpen] = useState(false);
  const reportBug = useReportBugDialog();
  const states = useActionStates([{ permission: "tickets:write", projectId }, { permission: "projects:write", projectId }]);
  if (states.every((state) => state.state === "hidden")) return null;

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
            projectId={projectId}
            verb="create tickets"
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
            projectId={projectId}
            verb="report bugs"
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
            projectId={projectId}
            verb="add a category"
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
