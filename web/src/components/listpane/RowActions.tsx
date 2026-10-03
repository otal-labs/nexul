import {
  CopyIcon,
  EyeIcon,
  EyeOffIcon,
  FolderInputIcon,
  LockIcon,
  LockOpenIcon,
  MoreHorizontalIcon,
  PencilIcon,
  PinIcon,
  PinOffIcon,
  Settings2Icon,
  Trash2Icon,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface RowActionsProps {
  itemLabel: string;
  /** Each action is hidden when its handler is omitted (the viewer lacks the permission). */
  /** Pin and Unpin are a personal view setting, so nothing gates them; hidden when omitted. */
  pin?: { pinned: boolean; onToggle: () => void } | undefined;
  lock?: { locked: boolean; onToggle: () => void } | undefined;
  /** Removing from the sidebar is personal too: the item stays, only the viewer's sidebar drops it. */
  sidebar?: { hidden: boolean; onToggle: () => void } | undefined;
  /** The places the item can move to, the current one checked. */
  moveTo?: { label: string; options: { id: string; name: string }[]; currentId: string; onMove: (id: string) => void } | undefined;
  onSettings?: (() => void) | undefined;
  onRename?: (() => void) | undefined;
  onClone?: (() => void) | undefined;
  onDelete?: (() => void) | undefined;
}

export const RowActions = ({ itemLabel, pin, lock, sidebar, moveTo, onSettings, onRename, onClone, onDelete }: RowActionsProps) => (
  <DropdownMenu>
    <DropdownMenuTrigger asChild>
      <Button variant="ghost" size="icon" className="size-7" aria-label={`More actions for ${itemLabel}`}>
        <MoreHorizontalIcon className="size-3.5" aria-hidden />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" className="min-w-36">
      {onSettings && (
        <DropdownMenuItem onSelect={onSettings}>
          <Settings2Icon aria-hidden />
          Settings
        </DropdownMenuItem>
      )}
      {pin && (
        <DropdownMenuItem onSelect={pin.onToggle}>
          {pin.pinned ? <PinOffIcon aria-hidden /> : <PinIcon aria-hidden />}
          {pin.pinned ? "Unpin" : "Pin"}
        </DropdownMenuItem>
      )}
      {sidebar && !sidebar.hidden && (
        <DropdownMenuItem onSelect={sidebar.onToggle}>
          <EyeOffIcon aria-hidden />
          Remove from sidebar
        </DropdownMenuItem>
      )}
      {sidebar?.hidden && (
        <DropdownMenuItem onSelect={sidebar.onToggle}>
          <EyeIcon aria-hidden />
          Show in sidebar
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
      {moveTo && (
        <DropdownMenuSub>
          <DropdownMenuSubTrigger>
            <FolderInputIcon aria-hidden />
            {moveTo.label}
          </DropdownMenuSubTrigger>
          <DropdownMenuSubContent className="min-w-36">
            <DropdownMenuRadioGroup value={moveTo.currentId} onValueChange={moveTo.onMove}>
              {moveTo.options.map((option) => (
                <DropdownMenuRadioItem key={option.id} value={option.id}>
                  {option.name}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuSubContent>
        </DropdownMenuSub>
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
      {onDelete && (!!onSettings || !!pin || !!sidebar || !!lock || !!moveTo || !!onRename || !!onClone) && <DropdownMenuSeparator />}
      {onDelete && (
        <DropdownMenuItem variant="destructive" onSelect={onDelete}>
          <Trash2Icon aria-hidden />
          Delete
        </DropdownMenuItem>
      )}
    </DropdownMenuContent>
  </DropdownMenu>
);
