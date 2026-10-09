import { SidebarFooter } from "@/components/sidebar/SidebarFooter";
import { SidebarHeader } from "@/components/sidebar/SidebarHeader";
import { SidebarNavContent } from "@/components/sidebar/SidebarNavContent";
import { TooltipProvider } from "@/components/ui/tooltip";
import { VoiceDock } from "@/components/voice/VoiceDock";
import { cn } from "@/lib/utils";

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapse: () => void;
  unreadCount: number;
}

// The rail is 68px so every icon keeps the x it has in the open sidebar: collapsing hides the labels and nothing else moves.
export const Sidebar = ({ collapsed, onToggleCollapse, unreadCount }: SidebarProps) => (
  <TooltipProvider delayDuration={400} skipDelayDuration={300}>
    <aside
      className={cn(
        "relative z-20 flex h-full shrink-0 flex-col",
        collapsed ? "w-17" : "w-60",
      )}
    >
      <SidebarHeader collapsed={collapsed} onToggleCollapse={onToggleCollapse} />
      <SidebarNavContent collapsed={collapsed} unreadCount={unreadCount} />
      <VoiceDock collapsed={collapsed} />
      <SidebarFooter collapsed={collapsed} />
    </aside>
  </TooltipProvider>
);
