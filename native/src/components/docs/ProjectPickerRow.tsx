import { Pressable } from "react-native";

import { Text } from "@/components/ui/text";
import type { Project } from "@/models/Project";

interface ProjectPickerRowProps {
  project: Project;
  onPress: (project: Project) => void;
}

export const ProjectPickerRow = ({ project, onPress }: ProjectPickerRowProps) => (
  <Pressable
    role="button"
    onPress={() => onPress(project)}
    className="min-h-11 flex-row items-center border-b border-border px-4 py-3 active:bg-accent"
  >
    <Text>{project.name}</Text>
  </Pressable>
);
