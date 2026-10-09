import { PickerRow } from "@/components/PickerRow";
import { ProjectMark } from "@/components/ProjectMark";
import type { Project } from "@/models/Project";

interface ProjectPickerRowProps {
  project: Project;
  selected: boolean;
  onPress: (project: Project) => void;
}

export const ProjectPickerRow = ({ project, selected, onPress }: ProjectPickerRowProps) => (
  <PickerRow
    label={project.name}
    selected={selected}
    onPress={() => onPress(project)}
    leading={<ProjectMark prefix={project.prefix} size="small" />}
  />
);
