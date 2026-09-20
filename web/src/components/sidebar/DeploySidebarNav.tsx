import { Cpu, Network, Workflow } from "lucide-react";
import { NavLink } from "react-router";

import { navLinkClass, sectionLabelClass, type SidebarNavEntry } from "@/components/SidebarNav";
import { cn } from "@/lib/utils";

const deployNav: SidebarNavEntry[] = [
  { to: "/runners", label: "Runners", icon: Cpu },
  { to: "/topology", label: "Topology", icon: Network, wip: true },
  { to: "/automations", label: "Automations", icon: Workflow },
];

interface DeploySidebarNavProps {
  collapsed: boolean;
}

export const DeploySidebarNav = ({ collapsed }: DeploySidebarNavProps) => (
  <>
    {!collapsed && <div className={sectionLabelClass}>Workspace</div>}
    <div className="flex flex-col gap-0.5">
      {deployNav.map((entry) => (
        <NavLink
          key={entry.to}
          to={entry.to}
          className={({ isActive }) => cn(navLinkClass({ isActive }), collapsed && "justify-center")}
          {...(entry.end ? { end: entry.end } : {})}
          {...(collapsed ? { title: entry.label } : {})}
        >
          <span className="flex w-8 shrink-0 justify-center">
            <entry.icon className="size-4" aria-hidden />
          </span>
          {!collapsed && <span className="flex-1 text-left">{entry.label}</span>}
          {!collapsed && entry.wip && (
            <span
              className="shrink-0 text-[10px] font-medium tracking-wide text-warning"
              title="Work in progress"
            >
              WIP
            </span>
          )}
        </NavLink>
      ))}
    </div>
  </>
);
