import type { ComponentType, ReactNode } from "react";

import * as Context from "@/components/ui/context-menu";
import * as Dropdown from "@/components/ui/dropdown-menu";
import type { ActionGate } from "@nexul/client-core/actionState";

import type { PanelEntity } from "@/models/PermissionPanel";

// The dropdown and context menu primitives share one shape, so a row's items render in either.
export interface MenuParts {
  Item: ComponentType<{
    children: ReactNode;
    onSelect?: (event: Event) => void;
    variant?: "default" | "destructive";
    className?: string;
    "aria-disabled"?: boolean;
    "aria-describedby"?: string;
  }>;
  Separator: ComponentType;
  Sub: ComponentType<{ children: ReactNode }>;
  SubTrigger: ComponentType<{ children: ReactNode }>;
  SubContent: ComponentType<{ children: ReactNode; className?: string }>;
  RadioGroup: ComponentType<{ children: ReactNode; value: string; onValueChange: (value: string) => void }>;
  RadioItem: ComponentType<{ children: ReactNode; value: string }>;
}

export const dropdownParts: MenuParts = {
  Item: Dropdown.DropdownMenuItem,
  Separator: Dropdown.DropdownMenuSeparator,
  Sub: Dropdown.DropdownMenuSub,
  SubTrigger: Dropdown.DropdownMenuSubTrigger,
  SubContent: Dropdown.DropdownMenuSubContent,
  RadioGroup: Dropdown.DropdownMenuRadioGroup,
  RadioItem: Dropdown.DropdownMenuRadioItem,
};

export const contextParts: MenuParts = {
  Item: Context.ContextMenuItem,
  Separator: Context.ContextMenuSeparator,
  Sub: Context.ContextMenuSub,
  SubTrigger: Context.ContextMenuSubTrigger,
  SubContent: Context.ContextMenuSubContent,
  RadioGroup: Context.ContextMenuRadioGroup,
  RadioItem: Context.ContextMenuRadioItem,
};

/** Each action is hidden when it is omitted (the viewer lacks the permission). */
export interface RowMenu {
  onSettings?: (() => void) | undefined;
  /** "Settings" unless the row names whose settings they are ("Project settings"). */
  settingsLabel?: string | undefined;
  /** A DM's or thread's bots, which have no settings page to live on. */
  onBots?: (() => void) | undefined;
  onNewDoc?: (() => void) | undefined;
  /** Pin and Unpin are a personal view setting, so nothing gates them. */
  pin?: { pinned: boolean; onToggle: () => void } | undefined;
  lock?: { locked: boolean; onToggle: () => void } | undefined;
  archive?: { archived: boolean; onToggle: () => void } | undefined;
  /** Removing from the sidebar is personal too: the item stays, only the viewer's sidebar drops it. */
  sidebar?: { hidden: boolean; onToggle: () => void } | undefined;
  /** The places the item can move to, the current one checked. */
  moveTo?: { label: string; options: { id: string; name: string }[]; currentId: string; onMove: (id: string) => void } | undefined;
  /** Puts a read notification back to unread; it is the viewer's own inbox, so nothing gates it. */
  onMarkUnread?: (() => void) | undefined;
  onEdit?: (() => void) | undefined;
  onRename?: (() => void) | undefined;
  onClone?: (() => void) | undefined;
  /** Shown only once the viewer may set the entity's rules; the row brings the opener so the panel stays out of the shell. */
  permissions?: { entity: PanelEntity; onOpen: () => void } | undefined;
  onDelete?: (() => void) | undefined;
  /** Draws Delete through the action state: locked with the reason when the viewer's role shows it. The row leaves onDelete out when the gate hides it. */
  deleteGate?: ActionGate | undefined;
}

// The row's own actions, before Permissions… and Delete.
export const hasOwnActions = (m: RowMenu) =>
  !!m.onSettings || !!m.onBots || !!m.onNewDoc || !!m.pin || !!m.sidebar || !!m.lock || !!m.archive || !!m.moveTo || !!m.onMarkUnread || !!m.onEdit || !!m.onRename || !!m.onClone;

// Permissions… counts only once its check has answered (manages), or a menu holding only it would open empty.
export const offersMenu = (m: RowMenu | undefined, manages = false): m is RowMenu =>
  !!m && (hasOwnActions(m) || !!m.onDelete || (manages && !!m.permissions));
