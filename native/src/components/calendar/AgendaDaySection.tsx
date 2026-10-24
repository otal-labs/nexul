import { View } from "react-native";

import { AgendaItemRow } from "@/components/calendar/AgendaItemRow";
import { Microheader } from "@/components/Microheader";
import type { AgendaDay, AgendaItem } from "@/lib/agenda";

const itemKey = (item: AgendaItem) => (item.kind === "due" ? `due:${item.ticket.id}` : `${item.occurrence.meeting_id}:${item.occurrence.slot}`);

export const AgendaDaySection = ({ day }: { day: AgendaDay }) => (
  <View className="pb-2">
    <Microheader className="px-5 pb-1.5 pt-4">{day.label}</Microheader>
    {day.items.map((item) => (
      <AgendaItemRow key={itemKey(item)} item={item} />
    ))}
  </View>
);
