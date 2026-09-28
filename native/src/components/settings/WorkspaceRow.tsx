import { Check } from "lucide-react-native";
import { Pressable } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import type { Workspace } from "@/models/Workspace";

interface WorkspaceRowProps {
  workspace: Workspace;
  selected: boolean;
  onPress: () => void;
}

export const WorkspaceRow = ({ workspace, selected, onPress }: WorkspaceRowProps) => {
  const [foreground] = useCSSVariable(["--color-foreground"]);
  return (
    <Pressable
      role="button"
      onPress={onPress}
      className="min-h-11 flex-row items-center justify-between gap-2 border-b border-border bg-card px-4 py-3 active:bg-accent"
    >
      <Text numberOfLines={1} className="flex-1 font-medium">
        {workspace.name}
      </Text>
      {selected && <Check size={16} color={String(foreground)} />}
    </Pressable>
  );
};
