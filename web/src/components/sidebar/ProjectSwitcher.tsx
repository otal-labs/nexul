import { useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { Popover } from "@/components/ui/popover";
import { NewDocButton } from "@/components/sidebar/NewDocButton";
import { ProjectSwitcherMenu } from "@/components/sidebar/ProjectSwitcherMenu";
import { ProjectSwitcherTrigger } from "@/components/sidebar/ProjectSwitcherTrigger";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { NEW_PROJECT_PATH, switchProjectPath, type Project } from "@/models/Project";

interface ProjectSwitcherProps {
  projects: Project[];
  current: Project;
  collapsed: boolean;
}

export const ProjectSwitcher = ({ projects, current, collapsed }: ProjectSwitcherProps) => {
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const selectProject = useWorkspaceStore((s) => s.selectProject);

  const handleSelect = (project: Project) => {
    setOpen(false);
    selectProject(project.id);
    void navigate(switchProjectPath(pathname, project));
  };

  // The project wizard's project step covers name + prefix; no dialog needed here.
  const handleCreate = () => {
    setOpen(false);
    void navigate(NEW_PROJECT_PATH);
  };

  return (
    <div className="group/project flex items-center">
      <Popover open={open} onOpenChange={setOpen}>
        <ProjectSwitcherTrigger current={current} collapsed={collapsed} />
        <ProjectSwitcherMenu
          projects={projects}
          currentId={current.id}
          onSelect={handleSelect}
          onCreate={handleCreate}
        />
      </Popover>
      {!collapsed && <NewDocButton project={current} />}
    </div>
  );
};
