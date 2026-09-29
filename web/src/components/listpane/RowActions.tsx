import { CopyIcon, MoreHorizontalIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";

interface RowActionsProps {
  itemLabel: string;
  /** Each action is hidden when its handler is omitted (the viewer lacks the permission). */
  onClone?: (() => void) | undefined;
  onDelete?: (() => void) | undefined;
}

export const RowActions = ({ itemLabel, onClone, onDelete }: RowActionsProps) => (
  <>
    {onClone && (
      <Button variant="ghost" size="icon" className="size-7" aria-label={`Clone ${itemLabel}`} title="Clone" onClick={onClone}>
        <CopyIcon className="size-3.5" aria-hidden />
      </Button>
    )}
    {onDelete && (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" className="size-7" aria-label={`More actions for ${itemLabel}`}>
            <MoreHorizontalIcon className="size-3.5" aria-hidden />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem variant="destructive" onSelect={onDelete}>
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    )}
  </>
);
