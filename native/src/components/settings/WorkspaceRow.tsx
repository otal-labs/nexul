import { GradientTile } from "@/components/GradientTile";
import { PickerRow } from "@/components/PickerRow";
import { Text } from "@/components/ui/text";
import type { Workspace } from "@/models/Workspace";

interface WorkspaceRowProps {
  workspace: Workspace;
  selected: boolean;
  onPress: () => void;
}

export const WorkspaceRow = ({ workspace, selected, onPress }: WorkspaceRowProps) => (
  <PickerRow
    label={workspace.name}
    selected={selected}
    onPress={onPress}
    leading={
      <GradientTile seed={workspace.id} className="size-7 rounded-md">
        <Text className="text-xs font-semibold text-white">{workspace.name.charAt(0).toUpperCase()}</Text>
      </GradientTile>
    }
  />
);
