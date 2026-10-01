import { CopyIcon, LockIcon, LockOpenIcon, MoreHorizontalIcon, PencilIcon, PinIcon, PinOffIcon, Trash2Icon } from "lucide-react";

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
  /** Pin and Unpin are a personal view setting, so nothing gates them; hidden when omitted. */
  pin?: { pinned: boolean; onToggle: () => void } | undefined;
  lock?: { locked: boolean; onToggle: () => void } | undefined;
  onRename?: (() => void) | undefined;
  onClone?: (() => void) | undefined;
  onDelete?: (() => void) | undefined;
}

export const RowActions = ({ itemLabel, pin, lock, onRename, onClone, onDelete }: RowActionsProps) => (
  <DropdownMenu>
    <DropdownMenuTrigger asChild>
      <Button variant="ghost" size="icon" className="size-7" aria-label={`More actions for ${itemLabel}`}>
        <MoreHorizontalIcon className="size-3.5" aria-hidden />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" className="min-w-36">
      {pin && (
        <DropdownMenuItem onSelect={pin.onToggle}>
          {pin.pinned ? <PinOffIcon aria-hidden /> : <PinIcon aria-hidden />}
          {pin.pinned ? "Unpin" : "Pin"}
        </DropdownMenuItem>
      )}
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
      {onRename && (
        <DropdownMenuItem onSelect={onRename}>
          <PencilIcon aria-hidden />
          Rename
        </DropdownMenuItem>
      )}
      {onClone && (
        <DropdownMenuItem onSelect={onClone}>
          <CopyIcon aria-hidden />
          Clone
        </DropdownMenuItem>
      )}
      {onDelete && (!!pin || !!lock || !!onRename || !!onClone) && <DropdownMenuSeparator />}
      {onDelete && (
        <DropdownMenuItem variant="destructive" onSelect={onDelete}>
          <Trash2Icon aria-hidden />
          Delete
        </DropdownMenuItem>
      )}
    </DropdownMenuContent>
  </DropdownMenu>
);
