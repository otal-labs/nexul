import { SectionList } from "react-native";

import type { SearchGroup, SearchHit } from "@nexul/client-core/search";

import { Microheader } from "@/components/Microheader";
import { SearchRow } from "@/components/search/SearchRow";
import { Text } from "@/components/ui/text";

interface SearchFeedProps {
  groups: SearchGroup[];
  cut: boolean;
  onSelect: (hit: SearchHit) => void;
}

export const SearchFeed = ({ groups, cut, onSelect }: SearchFeedProps) => (
  <SectionList
    sections={groups.map((group) => ({ title: group.heading, data: group.hits }))}
    keyExtractor={(hit) => `${hit.type}:${hit.id}`}
    keyboardShouldPersistTaps="handled"
    keyboardDismissMode="on-drag"
    stickySectionHeadersEnabled={false}
    contentContainerClassName="pb-6"
    renderSectionHeader={({ section }) => <Microheader className="px-5 pt-4 pb-1.5">{section.title}</Microheader>}
    renderItem={({ item }) => <SearchRow hit={item} onPress={onSelect} />}
    ListFooterComponent={cut ? <Text className="px-5 pt-4 text-[13px] text-muted-foreground">More matches than shown. Add a word to narrow.</Text> : null}
  />
);
