import { Inbox } from "lucide-react";
import { NavLink } from "react-router";

import { navLinkClass } from "@/components/SidebarNav";
import { cn } from "@/lib/utils";

interface SidebarInboxLinkProps {
  collapsed: boolean;
  unreadCount: number;
}

export const SidebarInboxLink = ({ collapsed, unreadCount }: SidebarInboxLinkProps) => (
  <NavLink
    to="/inbox"
    className={({ isActive }) => cn(navLinkClass({ isActive }), "relative", collapsed && "justify-center px-0")}
    {...(collapsed ? { title: "Inbox" } : {})}
  >
    <span className="flex w-8 shrink-0 justify-center">
      <Inbox className="size-4" aria-hidden />
    </span>
    {!collapsed && <span className="flex-1 text-left">Inbox</span>}
    {unreadCount > 0 && (
      <span
        className={cn(
          "flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground",
          collapsed && "absolute top-0 right-0",
        )}
      >
        {unreadCount > 99 ? "99+" : unreadCount}
      </span>
    )}
  </NavLink>
);
