import { HourglassIcon, RocketIcon } from "lucide-react";

import { RailTooltip } from "@/components/sidebar/RailTooltip";
import { SetupNavLink } from "@/components/sidebar/SetupNavLink";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Project } from "@/models/Project";
import { setupResumePath } from "@/models/ProjectWizard";
import { cn } from "@/lib/utils";

interface ProjectSetupNavProps {
  project: Project;
  collapsed: boolean;
}

const row = "flex items-center gap-2.5 rounded-md px-2.5 py-1 text-[13px]";

// While setup is open the project's one row is Continue setup, or Being set up for whoever can't change it (ADR 0143).
export const ProjectSetupNav = ({ project, collapsed }: ProjectSetupNavProps) => {
  const can = useAreaAccess(project.id);
  const wsPath = useWorkspacePath();
  const canSetUp = !!can?.("editProjects");

  return (
    <>
      {canSetUp && (
        <SetupNavLink to={wsPath(setupResumePath(project))} label="Continue setup" icon={RocketIcon} collapsed={collapsed} />
      )}
      {can && !canSetUp && (
        <RailTooltip label="Being set up" collapsed={collapsed}>
          <p className={cn(row, "text-muted-foreground")} aria-label={collapsed ? "Being set up" : undefined}>
            <span className="flex w-8 shrink-0 justify-center">
              <HourglassIcon className="size-4" aria-hidden />
            </span>
            {!collapsed && <span className="min-w-0 flex-1 truncate text-left">Being set up</span>}
          </p>
        </RailTooltip>
      )}
    </>
  );
};
