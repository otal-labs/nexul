import { CopyIcon, Ellipsis, Trash2Icon } from "lucide-react";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import type { Memory } from "@/models/Memory";

interface MemoryActionsMenuProps {
  memory: Memory;
  onClone: () => void;
  onDelete: () => void;
}

export const MemoryActionsMenu = ({ memory, onClone, onDelete }: MemoryActionsMenuProps) => (
  <Popover>
    <PopoverTrigger asChild>
      <Button variant="ghost" size="icon" className="size-7" aria-label="Memory actions">
        <Ellipsis className="size-4" />
      </Button>
    </PopoverTrigger>
    <PopoverContent align="end" className="w-48 p-1">
      <ActionMenuItem permission="memories:clone" projectId={memory.project_id} icon={CopyIcon} onSelect={onClone}>
        Clone to…
      </ActionMenuItem>
      <ActionMenuItem permission="memories:delete" projectId={memory.project_id} icon={Trash2Icon} destructive separated onSelect={onDelete}>
        Delete
      </ActionMenuItem>
    </PopoverContent>
  </Popover>
);
