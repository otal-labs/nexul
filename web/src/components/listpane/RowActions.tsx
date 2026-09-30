import { CopyIcon, LockIcon, LockOpenIcon, MoreHorizontalIcon, Trash2Icon } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface RowActionsProps {
  itemLabel: string;
  /** Each action is hidden when its handler is omitted (the viewer lacks the permission). */
  lock?: { locked: boolean; onToggle: () => void } | undefined;
  onClone?: (() => void) | undefined;
  onDelete?: (() => void) | undefined;
}

export const RowActions = ({ itemLabel, lock, onClone, onDelete }: RowActionsProps) => (
  <DropdownMenu>
    <DropdownMenuTrigger asChild>
      <Button variant="ghost" size="icon" className="size-7" aria-label={`More actions for ${itemLabel}`}>
        <MoreHorizontalIcon className="size-3.5" aria-hidden />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" className="min-w-36">
      {lock && !lock.locked && (
        <DropdownMenuItem onSelect={lock.onToggle}>
          <LockIcon aria-hidden />
          Lock
        </DropdownMenuItem>
      )}
      {lock?.locked && (
        <DropdownMenuItem onSelect={lock.onToggle}>
          <LockOpenIcon aria-hidden />
          Unlock
        </DropdownMenuItem>
      )}
      {onClone && (
        <DropdownMenuItem onSelect={onClone}>
          <CopyIcon aria-hidden />
          Clone
        </DropdownMenuItem>
      )}
      {onDelete && (!!lock || !!onClone) && <DropdownMenuSeparator />}
      {onDelete && (
        <DropdownMenuItem variant="destructive" onSelect={onDelete}>
          <Trash2Icon aria-hidden />
          Delete
        </DropdownMenuItem>
      )}
    </DropdownMenuContent>
  </DropdownMenu>
);
