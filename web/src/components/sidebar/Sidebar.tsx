import { SidebarFooter } from "@/components/sidebar/SidebarFooter";
import { SidebarHeader } from "@/components/sidebar/SidebarHeader";
import { SidebarNavContent } from "@/components/sidebar/SidebarNavContent";
import { VoiceDock } from "@/components/voice/VoiceDock";
import { cn } from "@/lib/utils";

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapse: () => void;
  unreadCount: number;
}

export const Sidebar = ({ collapsed, onToggleCollapse, unreadCount }: SidebarProps) => (
  <aside
    className={cn(
      "relative z-20 flex h-full shrink-0 flex-col",
      collapsed ? "w-14" : "w-60",
    )}
  >
    <SidebarHeader collapsed={collapsed} onToggleCollapse={onToggleCollapse} />
    <SidebarNavContent collapsed={collapsed} unreadCount={unreadCount} />
    <VoiceDock collapsed={collapsed} />
    <SidebarFooter collapsed={collapsed} />
  </aside>
);
