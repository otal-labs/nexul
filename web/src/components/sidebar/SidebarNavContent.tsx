import { SidebarChatLink } from "@/components/sidebar/SidebarChatLink";
import { ChatSidebarSection } from "@/components/sidebar/ChatSidebarSection";
import { DeploySidebarNav } from "@/components/sidebar/DeploySidebarNav";
import { ProjectTree } from "@/components/sidebar/ProjectTree";
import { SidebarInboxLink } from "@/components/sidebar/SidebarInboxLink";
import { WorkspaceSwitcher } from "@/components/WorkspaceSwitcher";

interface SidebarNavContentProps {
  collapsed: boolean;
  unreadCount: number;
}

export const SidebarNavContent = ({ collapsed, unreadCount }: SidebarNavContentProps) => (
  <>
    <div className="px-2 pb-1">
      <WorkspaceSwitcher collapsed={collapsed} />
    </div>
    <nav className="flex-1 overflow-y-auto px-2 py-1">
      <div className="flex flex-col gap-0.5">
        <SidebarInboxLink collapsed={collapsed} unreadCount={unreadCount} />
        <SidebarChatLink collapsed={collapsed} />
      </div>
      {collapsed && <div className="mx-2 my-2 border-t border-border/60" aria-hidden />}
      <ChatSidebarSection collapsed={collapsed} />
      {collapsed && <div className="mx-2 my-2 border-t border-border/60" aria-hidden />}
      <ProjectTree collapsed={collapsed} />
      {collapsed && <div className="mx-2 my-2 border-t border-border/60" aria-hidden />}
      <DeploySidebarNav collapsed={collapsed} />
    </nav>
  </>
);
