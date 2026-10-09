import { Inbox } from "lucide-react";
import { NavLink } from "react-router";

import { navLinkClass, railBadgeClass } from "@/components/SidebarNav";
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
  return (
    <RailTooltip label="Inbox" collapsed={collapsed}>
      <NavLink to={wsPath("/inbox")} aria-label={collapsed ? "Inbox" : undefined} className={navLinkClass}>
        <span className="flex w-8 shrink-0 justify-center">
          <Inbox className="size-4" aria-hidden />
        </span>
        {!collapsed && <span className={cn("flex-1 text-left", unreadCount > 0 && "font-medium text-foreground")}>Inbox</span>}
        <UnreadBadge count={unreadCount} className={cn(collapsed && railBadgeClass)} />
      </NavLink>
    </RailTooltip>
  );
};
