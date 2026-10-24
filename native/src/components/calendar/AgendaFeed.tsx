import type { ReactElement } from "react";
import { FlatList } from "react-native";

import { AgendaDaySection } from "@/components/calendar/AgendaDaySection";
import { RefreshList } from "@/components/RefreshList";
import type { AgendaDay } from "@/lib/agenda";

interface AgendaFeedProps {
  days: AgendaDay[];
  header: ReactElement;
  refreshing: boolean;
  onRefresh: () => void;
}

export const AgendaFeed = ({ days, header, refreshing, onRefresh }: AgendaFeedProps) => (
  <FlatList
    data={days}
    keyExtractor={(day) => day.key}
    ListHeaderComponent={header}
    contentContainerClassName="pb-6"
    renderItem={({ item }) => <AgendaDaySection day={item} />}
    refreshControl={<RefreshList refreshing={refreshing} onRefresh={onRefresh} />}
  />
);
