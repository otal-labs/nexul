import { ProjectMark } from "@/components/project/ProjectMark";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { SwitcherMenuItem } from "@/components/SwitcherMenu";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { projectEntity } from "@/models/PermissionPanel";
import { projectTile, type Project } from "@/models/Project";

interface ProjectSwitcherItemProps {
  project: Project;
  selected: boolean;
  onSelect: (project: Project) => void;
  onSettings: (project: Project) => void;
  onPermissions: (project: Project) => void;
}

// A project in the switcher; right-clicking it offers the project's settings and Permissions… to whoever may open them.
export const ProjectSwitcherItem = ({ project, selected, onSelect, onSettings, onPermissions }: ProjectSwitcherItemProps) => {
  const can = useAreaAccess(project.id);
  return (
    <RowContextMenu
      menu={{
        onSettings: can?.("projects") ? () => onSettings(project) : undefined,
        settingsLabel: "Project settings",
        permissions: { entity: projectEntity(project), onOpen: () => onPermissions(project) },
      }}
    >
      <div className="rounded-[5px]">
        <SwitcherMenuItem
          tile={projectTile(project)}
          mark={<ProjectMark project={project} className="h-5 w-auto min-w-6 rounded-sm px-1 text-[10px] shadow-none" />}
          name={project.name}
          selected={selected}
          onSelect={() => onSelect(project)}
        />
      </div>
    </RowContextMenu>
  );
};
