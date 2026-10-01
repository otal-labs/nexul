import type { ReactNode } from "react";
import { LockIcon, type LucideIcon } from "lucide-react";
import { NavLink } from "react-router";

import { navLinkClass } from "@/components/SidebarNav";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";

interface ChatSidebarRowProps {
  conversationId: string;
  label: string;
  icon: LucideIcon;
  // A private channel trails a muted lock.
  isPrivate?: boolean | undefined;
  unreadCount: number;
  onClick?: (() => void) | undefined;
  /** A … menu that takes the unread badge's place on hover, focus, and while open. */
  actions?: ReactNode;
}

// The sidebar is chat's only conversation list (ADR 0093), so the open conversation is the active row.
export const ChatSidebarRow = ({ conversationId, label, icon: Icon, isPrivate, unreadCount, onClick, actions }: ChatSidebarRowProps) => {
  const wsPath = useWorkspacePath();
  const revealed = "group-hover/row:hidden group-has-[:focus-visible]/row:hidden group-has-[[data-state=open]]/row:hidden";
  return (
    <div className="group/row relative">
      <NavLink
        to={wsPath(`/chat/${conversationId}`)}
        onClick={onClick}
        title={label}
        className={(state) =>
          cn(
            navLinkClass(state),
            !!actions && "group-hover/row:pr-9 group-has-[:focus-visible]/row:pr-9 group-has-[[data-state=open]]/row:pr-9",
          )
        }
      >
        <span className="flex w-8 shrink-0 justify-center">
          <Icon className="size-4" aria-hidden />
        </span>
        <span className="min-w-0 flex-1 truncate text-left">{label}</span>
        {isPrivate && <LockIcon role="img" className="size-3 shrink-0 text-muted-foreground/70" aria-label="Private" />}
        {unreadCount > 0 && (
          <span
            className={cn(
              "flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground",
              !!actions && revealed,
            )}
          >
            {unreadCount > 99 ? "99+" : unreadCount}
          </span>
        )}
      </NavLink>
      {!!actions && (
        <div className="absolute inset-y-0 right-1 hidden items-center group-has-[:focus-visible]/row:flex group-hover/row:flex has-[[data-state=open]]:flex">
          {actions}
        </div>
      )}
    </div>
  );
};
