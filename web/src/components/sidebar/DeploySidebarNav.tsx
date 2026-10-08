import { ChevronDownIcon, Cpu, Network, SlidersHorizontal, Workflow } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { sectionLabelClass, type SidebarNavEntry } from "@/components/SidebarNav";
import { SidebarNavLink } from "@/components/sidebar/SidebarNavLink";
import { useCanOpen } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import { useSidebarStore } from "@/stores/sidebarStore";

const deployNav: SidebarNavEntry[] = [
  { to: "/runners", label: "Runners", icon: Cpu, area: "runners" },
  { to: "/topology", label: "Topology", icon: Network, area: "topology" },
  { to: "/automations", label: "Automations", icon: Workflow, area: "automations", wip: true },
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
  const entries = deployNav.filter((entry) => canOpen(entry.area) === true);

  if (entries.length === 0) return null;

  return (
    <div role="group" aria-label="Workspace" className="flex flex-col gap-0.5">
      {collapsed && <div className="mx-2 my-2 border-t border-border" aria-hidden />}
      {!collapsed && (
        <button
          type="button"
          onClick={toggle}
          aria-expanded={open}
          className={cn(
            sectionLabelClass,
            "flex w-full items-center justify-between rounded-md outline-none transition-colors duration-150 ease-standard hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/40",
          )}
        >
          <span>Workspace</span>
          <ChevronDownIcon
            className={cn("size-3.5 transition-transform duration-150 ease-standard", !open && "-rotate-90")}
            aria-hidden
          />
        </button>
      )}
      {(open || collapsed) && (
        <div className="flex flex-col gap-0.5">
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
