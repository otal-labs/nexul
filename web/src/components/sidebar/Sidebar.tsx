import { Link } from "react-router";

import { CollapseToggleButton } from "@/components/sidebar/CollapseToggleButton";
import { SidebarFooter } from "@/components/sidebar/SidebarFooter";
import { SidebarNavContent } from "@/components/sidebar/SidebarNavContent";
import { Logo } from "@/components/Logo";
import { VoiceDock } from "@/components/voice/VoiceDock";
import { cn } from "@/lib/utils";

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapse: () => void;
  isLoggedIn: boolean;
  unreadCount: number;
}

export const Sidebar = ({ collapsed, onToggleCollapse, isLoggedIn, unreadCount }: SidebarProps) => (
  <div className="relative shrink-0">
    <aside
      className={cn(
        "sticky top-0 flex h-screen flex-col border-r border-border bg-surface-2 transition-[width] duration-200 ease-standard",
        collapsed ? "w-14" : "w-60",
      )}
    >
      <div className={cn("flex h-14 items-center gap-2.5 px-[18px]", collapsed && "justify-center px-0")}>
        <Link to="/" className="flex min-w-0 items-center gap-2.5 font-semibold tracking-tight">
          <Logo />
          {!collapsed && <span className="text-[15px]">Nexul</span>}
        </Link>
        {!collapsed && (
          <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
            Beta
          </span>
        )}
      </div>
      {isLoggedIn && <SidebarNavContent collapsed={collapsed} unreadCount={unreadCount} />}
      {!isLoggedIn && <div className="flex-1" />}
      {isLoggedIn && <VoiceDock collapsed={collapsed} />}
      <SidebarFooter collapsed={collapsed} isLoggedIn={isLoggedIn} />
    </aside>
    <CollapseToggleButton collapsed={collapsed} onToggle={onToggleCollapse} />
  </div>
);
