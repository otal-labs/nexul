import { MessagesSquareIcon } from "lucide-react";
import { NavLink } from "react-router";

import { navLinkClass, railDotClass } from "@/components/SidebarNav";
import { cn } from "@/lib/utils";
import { RailTooltip } from "@/components/sidebar/RailTooltip";
import { useFetchChatUnread } from "@/hooks/ChatHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The icon rail has no room for the conversation list, so chat collapses to one way in with a dot for anything unread.
export const SidebarChatRailLink = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const wsPath = useWorkspacePath();
  const { data: unread } = useFetchChatUnread(workspaceId);
  const waiting = Object.values(unread ?? {}).some((count) => count > 0);
  return (
    <RailTooltip label={waiting ? "Chat, unread messages" : "Chat"} collapsed>
      <NavLink to={wsPath("/chat")} className={navLinkClass}>
        <span className="flex w-8 shrink-0 justify-center">
          <MessagesSquareIcon className="size-4" aria-hidden />
        </span>
        <span className="sr-only">{waiting ? "Chat, unread messages" : "Chat"}</span>
        {waiting && <span aria-hidden className={cn(railDotClass, "bg-foreground")} />}
      </NavLink>
    </RailTooltip>
  );
};
