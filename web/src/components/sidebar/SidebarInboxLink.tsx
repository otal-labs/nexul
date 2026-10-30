import { Inbox } from "lucide-react";
import { NavLink } from "react-router";

import { navLinkClass, railDotClass } from "@/components/SidebarNav";
import { RailTooltip } from "@/components/sidebar/RailTooltip";
import { UnreadBadge } from "@/components/UnreadBadge";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";

interface SidebarInboxLinkProps {
  collapsed: boolean;
  unreadCount: number;
}

export const SidebarInboxLink = ({ collapsed, unreadCount }: SidebarInboxLinkProps) => {
  const wsPath = useWorkspacePath();
  const railLabel = unreadCount > 0 ? `Inbox, ${unreadCount} unread` : "Inbox";
  return (
    <RailTooltip label={railLabel} collapsed={collapsed}>
      <NavLink to={wsPath("/inbox")} aria-label={collapsed ? railLabel : undefined} className={navLinkClass}>
        <span className="flex w-8 shrink-0 justify-center">
          <Inbox className="size-4" aria-hidden />
        </span>
        {!collapsed && <span className={cn("flex-1 text-left", unreadCount > 0 && "font-medium text-foreground")}>Inbox</span>}
        {!collapsed && <UnreadBadge count={unreadCount} />}
        {collapsed && unreadCount > 0 && <span aria-hidden className={cn(railDotClass, "bg-foreground")} />}
      </NavLink>
    </RailTooltip>
  );
};
