import { FlatList } from "react-native";

import { StackRow } from "@/components/deploys/StackRow";
import type { StackWithLatestDeploy } from "@/hooks/StackHooks";

interface StackFeedProps {
  rows: StackWithLatestDeploy[];
  onSelect: (id: string) => void;
}

export const StackFeed = ({ rows, onSelect }: StackFeedProps) => (
  <FlatList
    data={rows}
    keyExtractor={(row) => row.stack.id}
    renderItem={({ item }) => <StackRow stack={item.stack} latest={item.latest} onPress={() => onSelect(item.stack.id)} />}
  />
);
