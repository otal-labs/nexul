import type { LucideIcon } from "lucide-react";
import { NavLink } from "react-router";

import { navLinkClass } from "@/components/SidebarNav";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

interface ChatSidebarRowProps {
  conversationId: string;
  label: string;
  icon: LucideIcon;
  unreadCount: number;
  onClick?: (() => void) | undefined;
}

// The sidebar is chat's only conversation list (ADR 0093), so the open conversation is the active row.
export const ChatSidebarRow = ({ conversationId, label, icon: Icon, unreadCount, onClick }: ChatSidebarRowProps) => {
  const wsPath = useWorkspacePath();
  return (
    <NavLink to={wsPath(`/chat/${conversationId}`)} onClick={onClick} title={label} className={navLinkClass}>
      <span className="flex w-8 shrink-0 justify-center">
        <Icon className="size-4" aria-hidden />
      </span>
      <span className="min-w-0 flex-1 truncate text-left">{label}</span>
      {unreadCount > 0 && (
        <span className="flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
          {unreadCount > 99 ? "99+" : unreadCount}
        </span>
      )}
    </NavLink>
  );
};
