import { CommandPaletteTrigger } from "@/components/command/CommandPaletteTrigger";
import { ChatSidebarSection } from "@/components/sidebar/ChatSidebarSection";
import { SidebarActiveIndicator } from "@/components/sidebar/SidebarActiveIndicator";
import { DeploySidebarNav } from "@/components/sidebar/DeploySidebarNav";
import { ProjectSection } from "@/components/sidebar/ProjectSection";
import { SidebarChatRailLink } from "@/components/sidebar/SidebarChatRailLink";
import { SidebarInboxLink } from "@/components/sidebar/SidebarInboxLink";
import { WorkspaceSwitcher } from "@/components/WorkspaceSwitcher";

interface SidebarNavContentProps {
  collapsed: boolean;
  unreadCount: number;
}

// Places before conversations: the project's and the workspace's fixed pages lead, so a long channel list never pushes
// them below the fold.
export const SidebarNavContent = ({ collapsed, unreadCount }: SidebarNavContentProps) => (
  <>
    <div className="px-2 pb-1">
      <WorkspaceSwitcher collapsed={collapsed} />
    </div>
    <nav aria-label="Main" className="scroll-edge relative isolate flex-1 overflow-y-auto px-2 pt-1 pb-6">
      <SidebarActiveIndicator />
      <div className="flex flex-col gap-0.5">
        <CommandPaletteTrigger collapsed={collapsed} />
        <SidebarInboxLink collapsed={collapsed} unreadCount={unreadCount} />
        {collapsed && <SidebarChatRailLink />}
      </div>
      <ProjectSection collapsed={collapsed} />
      <DeploySidebarNav collapsed={collapsed} />
      <ChatSidebarSection collapsed={collapsed} />
    </nav>
  </>
);
