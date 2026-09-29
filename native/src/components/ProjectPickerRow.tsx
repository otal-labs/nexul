import { Check } from "lucide-react-native";
import { Pressable } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import type { Project } from "@/models/Project";

interface ProjectPickerRowProps {
  project: Project;
  selected: boolean;
  onPress: (project: Project) => void;
}

export const ProjectPickerRow = ({ project, selected, onPress }: ProjectPickerRowProps) => {
  const [foreground] = useCSSVariable(["--color-foreground"]);
  return (
    <Pressable
      role="button"
      aria-selected={selected}
      onPress={() => onPress(project)}
      className="min-h-11 flex-row items-center gap-2 border-b border-border px-4 py-3 active:bg-accent"
    >
      <Text className="min-w-0 flex-1 font-medium" numberOfLines={1}>
        {project.name}
      </Text>
      <Text variant="small" className="shrink-0 font-mono text-muted-foreground">
        {project.prefix}
      </Text>
      {selected && <Check size={16} color={String(foreground)} />}
    </Pressable>
  );
};
