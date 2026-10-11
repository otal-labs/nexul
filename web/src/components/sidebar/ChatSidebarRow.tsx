import type { ReactNode } from "react";
import { LockIcon, type LucideIcon } from "lucide-react";
import { NavLink } from "react-router";

import { RowActions } from "@/components/listpane/RowActions";
import { offersMenu, type RowMenu } from "@/components/listpane/rowMenu";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { navLinkClass } from "@/components/SidebarNav";
import { UnreadBadge } from "@/components/UnreadBadge";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";

interface ChatSidebarRowProps {
  conversationId: string;
  label: string;
  icon: LucideIcon;
  // Stands in for the icon: a direct message shows the person's avatar.
  leading?: ReactNode;
  // A private channel's lock takes the place of its glyph.
  isPrivate?: boolean | undefined;
  unreadCount: number;
  onClick?: (() => void) | undefined;
  /** A … menu that takes the unread badge's place on hover, focus, and while open. */
  actions?: ReactNode;
  /** The row's menu, as its … and on right-click; a row with a menu passes no actions. */
  menu?: RowMenu | undefined;
  /** What the … names, when the row reads differently (#general). */
  menuLabel?: string | undefined;
}

// The sidebar is chat's only conversation list (ADR 0093), so the open conversation is the active row.
export const ChatSidebarRow = ({ conversationId, label, icon: Icon, leading, isPrivate, unreadCount, onClick, actions, menu, menuLabel = label }: ChatSidebarRowProps) => {
  const wsPath = useWorkspacePath();
  const hasActions = !!actions || offersMenu(menu);
  const revealed = "group-hover/row:hidden group-has-[:focus-visible]/row:hidden group-has-[[data-state=open]]/row:hidden";
  return (
    <RowContextMenu menu={menu}>
      <div className="group/row relative rounded-md">
        <NavLink
          to={wsPath(`/chat/${conversationId}`)}
          onClick={onClick}
          title={label}
          className={(state) =>
            cn(
              navLinkClass(state),
              hasActions && "group-hover/row:pr-9 group-has-[:focus-visible]/row:pr-9 group-has-[[data-state=open]]/row:pr-9",
            )
          }
        >
          <span className="flex w-8 shrink-0 justify-center">
            {leading}
            {!leading && isPrivate && <LockIcon role="img" className="size-4" aria-label="Private" />}
            {!leading && !isPrivate && <Icon className="size-4" aria-hidden />}
          </span>
          <span className={cn("min-w-0 flex-1 truncate text-left", unreadCount > 0 && "font-medium text-foreground")}>{label}</span>
          <UnreadBadge count={unreadCount} className={hasActions ? revealed : undefined} />
        </NavLink>
        {hasActions && (
          <div className="absolute inset-y-0 right-1 hidden items-center group-has-[:focus-visible]/row:flex group-hover/row:flex has-[[data-state=open]]:flex">
            {actions}
            {offersMenu(menu) && <RowActions itemLabel={menuLabel} {...menu} />}
          </div>
        )}
      </div>
    </RowContextMenu>
  );
};
