import { MessageSquare } from "lucide-react";
import { NavLink } from "react-router";

import { navLinkClass } from "@/components/SidebarNav";
import { useFetchChatUnread } from "@/hooks/ChatHooks";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface SidebarChatLinkProps {
  collapsed: boolean;
}

export const SidebarChatLink = ({ collapsed }: SidebarChatLinkProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: unread } = useFetchChatUnread(workspaceId);
  const unreadTotal = unread ? Object.values(unread).reduce((sum, n) => sum + n, 0) : 0;

  return (
    <NavLink
      to="/chat"
      className={({ isActive }) => cn(navLinkClass({ isActive }), "relative", collapsed && "justify-center px-0")}
      {...(collapsed ? { title: "Chat" } : {})}
    >
      <span className="flex w-8 shrink-0 justify-center">
        <MessageSquare className="size-4" aria-hidden />
      </span>
      {!collapsed && <span className="flex-1 text-left">Chat</span>}
      {unreadTotal > 0 && (
        <span
          className={cn(
            "flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground",
            collapsed && "absolute top-0 right-0",
          )}
        >
          {unreadTotal > 99 ? "99+" : unreadTotal}
        </span>
      )}
    </NavLink>
  );
};
