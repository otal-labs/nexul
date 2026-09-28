import { FlatList, RefreshControl } from "react-native";
import { useCSSVariable } from "uniwind";

import { DocRow } from "@/components/docs/DocRow";
import type { DocListItem } from "@/models/Doc";

interface DocsFeedProps {
  docs: DocListItem[];
  refreshing: boolean;
  onRefresh: () => void;
  onSelect: (doc: DocListItem) => void;
}

export const DocsFeed = ({ docs, refreshing, onRefresh, onSelect }: DocsFeedProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <FlatList
      data={docs}
      keyExtractor={(doc) => doc.id}
      renderItem={({ item }) => <DocRow doc={item} onPress={onSelect} />}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={String(mutedForeground)} />
      }
    />
  );
};
