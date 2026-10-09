import { useLayoutEffect, useRef } from "react";

import { SidebarFooter } from "@/components/sidebar/SidebarFooter";
import { SidebarHeader } from "@/components/sidebar/SidebarHeader";
import { SidebarNavContent } from "@/components/sidebar/SidebarNavContent";
import { TooltipProvider } from "@/components/ui/tooltip";
import { VoiceDock } from "@/components/voice/VoiceDock";
import { revealSidebar } from "@/lib/motion";
import { cn } from "@/lib/utils";

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapse: () => void;
  unreadCount: number;
}

// The rail is 68px so every icon keeps the x it has in the open sidebar: collapsing hides the labels and nothing else
// moves, and opening wipes them in over a layout that has already snapped.
export const Sidebar = ({ collapsed, onToggleCollapse, unreadCount }: SidebarProps) => {
  const ref = useRef<HTMLElement>(null);
  const mounted = useRef(false);
  useLayoutEffect(() => {
    if (!mounted.current) {
      mounted.current = true;
      return;
    }
    revealSidebar(ref.current, !collapsed);
  }, [collapsed]);
  return (
    <TooltipProvider delayDuration={400} skipDelayDuration={300}>
      <aside
        ref={ref}
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
};
