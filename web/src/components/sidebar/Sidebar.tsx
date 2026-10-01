import { SidebarFooter } from "@/components/sidebar/SidebarFooter";
import { SidebarHeader } from "@/components/sidebar/SidebarHeader";
import { SidebarNavContent } from "@/components/sidebar/SidebarNavContent";
import { VoiceDock } from "@/components/voice/VoiceDock";
import { cn } from "@/lib/utils";

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapse: () => void;
  isLoggedIn: boolean;
  unreadCount: number;
}

export const Sidebar = ({ collapsed, onToggleCollapse, isLoggedIn, unreadCount }: SidebarProps) => (
  <aside
    className={cn(
      "sticky top-0 z-20 flex h-screen shrink-0 flex-col border-r border-border bg-surface-2",
      collapsed ? "w-14" : "w-60",
    )}
  >
    <SidebarHeader collapsed={collapsed} isLoggedIn={isLoggedIn} onToggleCollapse={onToggleCollapse} />
    {isLoggedIn && <SidebarNavContent collapsed={collapsed} unreadCount={unreadCount} />}
    {!isLoggedIn && <div className="flex-1" />}
    {isLoggedIn && <VoiceDock collapsed={collapsed} />}
    {isLoggedIn && <SidebarFooter collapsed={collapsed} />}
  </aside>
);
