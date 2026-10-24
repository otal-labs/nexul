import { CalendarDays, Cpu, Network, ShieldCheck, SlidersHorizontal, Sparkles, Workflow } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { type SidebarNavEntry } from "@/components/SidebarNav";
import { SidebarFoldHeader } from "@/components/sidebar/SidebarFoldHeader";
import { SidebarNavLink } from "@/components/sidebar/SidebarNavLink";
import { useCanOpen, useShowRunnersLink } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useSidebarStore } from "@/stores/sidebarStore";

const deployNav: SidebarNavEntry[] = [
  { to: "/calendar", label: "Calendar", icon: CalendarDays },
  { to: "/runners", label: "Runners", icon: Cpu },
  { to: "/topology", label: "Topology", icon: Network, area: "topology" },
  { to: "/plays", label: "Plays", icon: Sparkles, area: "plays" },
  { to: "/automations", label: "Automations", icon: Workflow, area: "automations", wip: true },
  { to: "/access", label: "Access", icon: ShieldCheck, area: "access" },
  { to: "/configuration", label: "Configuration", icon: SlidersHorizontal, area: "configuration" },
];

interface DeploySidebarNavProps {
  collapsed: boolean;
}

// The workspace's own pages, under the project's; folding it leaves only its header.
export const DeploySidebarNav = ({ collapsed }: DeploySidebarNavProps) => {
  const { open, toggle } = useSidebarStore(
    useShallow((s) => ({ open: s.workspaceNavOpen, toggle: s.toggleWorkspaceNav })),
  );
  const canOpen = useCanOpen();
  const wsPath = useWorkspacePath();
  const showRunners = useShowRunnersLink();
  const entries = deployNav.filter((entry) => (entry.to === "/runners" ? showRunners : !entry.area || canOpen(entry.area) === true));

  if (entries.length === 0) return null;

  return (
    <div role="group" aria-label="Workspace" className="flex flex-col gap-0.5">
      {collapsed && <div className="mx-2 my-2 border-t border-border" aria-hidden />}
      {!collapsed && <SidebarFoldHeader label="Workspace" open={open} onToggle={toggle} />}
      {(open || collapsed) && (
        <div data-fold className="flex flex-col gap-0.5">
          {entries.map((entry) => (
            <SidebarNavLink
              key={entry.to}
              to={wsPath(entry.to)}
              label={entry.label}
              icon={entry.icon}
              collapsed={collapsed}
              end={entry.end ?? false}
            >
              {entry.wip && (
                <span className="shrink-0 font-mono text-xs text-warning" title="Work in progress">
                  WIP
                </span>
              )}
            </SidebarNavLink>
          ))}
        </div>
      )}
    </div>
  );
};
