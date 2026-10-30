import type { ComponentType } from "react";
import { NavLink } from "react-router";

import { navLinkClass, railDotClass } from "@/components/SidebarNav";
import { RailTooltip } from "@/components/sidebar/RailTooltip";
import { cn } from "@/lib/utils";

interface SetupNavLinkProps {
  to: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  collapsed: boolean;
}

// A plain nav row; the warning dot says setup is still open, trailing the label or on the rail icon's corner.
export const SetupNavLink = ({ to, label, icon: Icon, collapsed }: SetupNavLinkProps) => (
  <RailTooltip label={label} collapsed={collapsed}>
    <NavLink to={to} aria-label={collapsed ? `${label}, setup pending` : undefined} className={navLinkClass}>
      <span className="flex w-8 shrink-0 justify-center">
        <Icon className="size-4" aria-hidden />
      </span>
      {!collapsed && <span className="min-w-0 flex-1 truncate text-left">{label}</span>}
      <span
        aria-hidden
        className={cn("size-1.5 shrink-0 rounded-full bg-warning", collapsed && railDotClass)}
      />
    </NavLink>
  </RailTooltip>
);
