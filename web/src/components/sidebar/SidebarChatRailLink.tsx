import { MessagesSquareIcon } from "lucide-react";
import { NavLink } from "react-router";

import { navLinkClass } from "@/components/SidebarNav";
import { useFetchChatUnread } from "@/hooks/ChatHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { cn } from "@/lib/utils";

// The icon rail has no room for the conversation list, so chat collapses to one way in with a dot for anything unread.
export const SidebarChatRailLink = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const wsPath = useWorkspacePath();
  const { data: unread } = useFetchChatUnread(workspaceId);
  const waiting = Object.values(unread ?? {}).some((count) => count > 0);
  return (
    <NavLink to={wsPath("/chat")} title="Chat" className={({ isActive }) => cn(navLinkClass({ isActive }), "relative justify-center px-0")}>
      <span className="flex w-8 shrink-0 justify-center">
        <MessagesSquareIcon className="size-4" aria-hidden />
      </span>
      <span className="sr-only">{waiting ? "Chat, unread messages" : "Chat"}</span>
      {waiting && <span aria-hidden className="absolute top-1.5 right-2.5 size-1.5 rounded-full bg-foreground" />}
    </NavLink>
  );
};
