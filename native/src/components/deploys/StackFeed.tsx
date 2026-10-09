import { FlatList } from "react-native";

import { StackRow } from "@/components/deploys/StackRow";
import { ScreenHeader } from "@/components/ScreenHeader";
import type { StackWithLatestDeploy } from "@/hooks/StackHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { DeployStatus } from "@/models/Stack";

interface StackFeedProps {
  rows: StackWithLatestDeploy[];
  onSelect: (id: string) => void;
}

const fleetLine = (rows: StackWithLatestDeploy[]) => {
  const failing = rows.filter((r) => r.latest?.status === DeployStatus.Failed).length;
  const active = rows.filter((r) => r.latest?.status === DeployStatus.Running || r.latest?.status === DeployStatus.Pending).length;
  return [`${rows.length} ${rows.length === 1 ? "stack" : "stacks"}`, active > 0 && `${active} deploying`, failing > 0 && `${failing} failed`]
    .filter(Boolean)
    .join(" · ");
};

export const StackFeed = ({ rows, onSelect }: StackFeedProps) => {
  const workspace = useSelectedWorkspace();
  return (
    <FlatList
      data={rows}
      keyExtractor={(row) => row.stack.id}
      ListHeaderComponent={<ScreenHeader eyebrow={workspace?.name} title="Deploys" meta={fleetLine(rows)} />}
      contentContainerClassName="pb-6"
      renderItem={({ item }) => <StackRow stack={item.stack} latest={item.latest} onPress={() => onSelect(item.stack.id)} />}
    />
  );
};
