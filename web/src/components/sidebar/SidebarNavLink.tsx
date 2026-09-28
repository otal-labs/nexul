import type { ComponentType, ReactNode } from "react";
import { NavLink } from "react-router";

import { navLinkClass } from "@/components/SidebarNav";
import { cn } from "@/lib/utils";

interface SidebarNavLinkProps {
  to: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  collapsed: boolean;
  end?: boolean;
  children?: ReactNode;
}

// children trail the label (a WIP tag, a count) and hide with it on the collapsed rail.
export const SidebarNavLink = ({ to, label, icon: Icon, collapsed, end = false, children }: SidebarNavLinkProps) => (
  <NavLink
    to={to}
    end={end}
    title={collapsed ? label : undefined}
    className={({ isActive }) => cn(navLinkClass({ isActive }), collapsed && "justify-center px-0")}
  >
    <span className="flex w-8 shrink-0 justify-center">
      <Icon className="size-4" aria-hidden />
    </span>
    {!collapsed && <span className="min-w-0 flex-1 truncate text-left">{label}</span>}
    {!collapsed && children}
  </NavLink>
);
