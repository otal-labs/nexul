import { useState } from "react";
import { useNavigate } from "react-router";

import { Popover } from "@/components/ui/popover";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { ProjectMark } from "@/components/project/ProjectMark";
import { ProjectSwitcherMenu } from "@/components/sidebar/ProjectSwitcherMenu";
import { SwitcherTrigger } from "@/components/SwitcherTrigger";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { useWorkspacePath, useWorkspacePathname } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { projectEntity } from "@/models/PermissionPanel";
import { NEW_PROJECT_PATH, projectSettingsPath, projectTile, projectToken, switchProjectPath, type Project } from "@/models/Project";

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
  const canOpenCurrent = useAreaAccess(current.id);
  const openPermissions = usePermissionsDialog();
  const currentEntity = projectEntity(current);

  const handleSelect = (project: Project) => {
    setOpen(false);
    selectProject(project.id);
    void navigate(wsPath(switchProjectPath(pathname, project)));
  };

  const handleSettings = (project: Project) => {
    setOpen(false);
    selectProject(project.id);
    void navigate(wsPath(projectSettingsPath(projectToken(project))));
  };

  const handlePermissions = (project: Project) => {
    setOpen(false);
    void openPermissions(projectEntity(project));
  };

  // The project wizard's project step covers name + prefix; no dialog needed here.
  const handleCreate = () => {
    setOpen(false);
    void navigate(wsPath(NEW_PROJECT_PATH));
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <RowContextMenu
        menu={{
          onSettings: canOpenCurrent?.("projects") ? () => handleSettings(current) : undefined,
          settingsLabel: "Project settings",
          permissions: { entity: currentEntity, onOpen: () => handlePermissions(current) },
        }}
      >
        <div className="rounded-md">
          <SwitcherTrigger
            tile={projectTile(current)}
            name={current.name}
            collapsed={collapsed}
            mark={<ProjectMark project={current} className="size-8 rounded-lg text-[11px] shadow-none" />}
          />
        </div>
      </RowContextMenu>
      <ProjectSwitcherMenu
        projects={projects}
        currentId={current.id}
        onSelect={handleSelect}
        onSettings={handleSettings}
        onPermissions={handlePermissions}
        onCreate={handleCreate}
        collapsed={collapsed}
      />
    </Popover>
  );
};
