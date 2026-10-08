import { useState } from "react";
import { useNavigate } from "react-router";

import { Popover } from "@/components/ui/popover";
import { ProjectMark } from "@/components/project/ProjectMark";
import { ProjectSwitcherMenu } from "@/components/sidebar/ProjectSwitcherMenu";
import { SwitcherTrigger } from "@/components/SwitcherTrigger";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath, useWorkspacePathname } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { NEW_PROJECT_PATH, projectTile, switchProjectPath, type Project } from "@/models/Project";

interface ProjectSwitcherProps {
  projects: Project[];
  current: Project;
  collapsed: boolean;
}

export const ProjectSwitcher = ({ projects, current, collapsed }: ProjectSwitcherProps) => {
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const pathname = useWorkspacePathname();
  const selectProject = useWorkspaceStore((s) => s.selectProject);
  const can = useAreaAccess();

  const handleSelect = (project: Project) => {
    setOpen(false);
    selectProject(project.id);
    void navigate(wsPath(switchProjectPath(pathname, project)));
  };

  // The project wizard's project step covers name + prefix; no dialog needed here.
  const handleCreate = () => {
    setOpen(false);
    void navigate(wsPath(NEW_PROJECT_PATH));
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <SwitcherTrigger
        tile={projectTile(current)}
        name={current.name}
        collapsed={collapsed}
        mark={<ProjectMark project={current} className="h-7 w-auto min-w-8 rounded-md px-1 text-[11px] shadow-none" />}
      />
      <ProjectSwitcherMenu
        projects={projects}
        currentId={current.id}
        onSelect={handleSelect}
        onCreate={can?.("newProject") ? handleCreate : undefined}
      />
    </Popover>
  );
};
