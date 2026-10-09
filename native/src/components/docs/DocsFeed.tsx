import type { ReactElement } from "react";
import { FlatList } from "react-native";

import { DocRow } from "@/components/docs/DocRow";
import { RefreshList } from "@/components/RefreshList";
import type { DocListItem } from "@/models/Doc";

interface DocsFeedProps {
  docs: DocListItem[];
  header: ReactElement;
  refreshing: boolean;
  onRefresh: () => void;
  onSelect: (doc: DocListItem) => void;
}

export const DocsFeed = ({ docs, header, refreshing, onRefresh, onSelect }: DocsFeedProps) => (
  <FlatList
    data={docs}
    keyExtractor={(doc) => doc.id}
    ListHeaderComponent={header}
    contentContainerClassName="pb-6"
    renderItem={({ item }) => <DocRow doc={item} onPress={onSelect} />}
    refreshControl={<RefreshList refreshing={refreshing} onRefresh={onRefresh} />}
  />
);
