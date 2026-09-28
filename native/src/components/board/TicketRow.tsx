import { Pressable, View } from "react-native";

import { Text } from "@/components/ui/text";
import { labelDotColor, ticketTypeDotColor } from "@/components/board/ticketTypeColor";
import { cn } from "@/lib/utils";
import { ticketKey, type Ticket } from "@/models/Ticket";

interface TicketRowProps {
  ticket: Ticket;
  projectPrefix: string | undefined;
  typeName: string | undefined;
  typeColor: string | undefined;
  onPress: () => void;
}

// Hairline row per practices/native.md: primary field (title) left, mono meta (the ticket key) right.
// Type and labels render as dots only here (ticket 28); the ticket screen shows the type's name in full.
export const TicketRow = ({ ticket, projectPrefix, typeName, typeColor, onPress }: TicketRowProps) => {
  const labels = ticket.labels ?? [];
  return (
    <Pressable
      role="button"
      onPress={onPress}
      className="min-h-11 flex-row items-center gap-2 border-b border-border px-4 py-2.5 active:bg-accent"
    >
      <View className="flex-row items-center gap-1" accessible aria-label={typeName ? `Type: ${typeName}` : undefined}>
        {typeName && <View className={cn("size-2 rounded-full", ticketTypeDotColor(typeName, typeColor))} />}
        {labels.map((label) => (
          <View key={label} accessible aria-label={`Label: ${label}`} className={cn("size-2 rounded-full", labelDotColor(label))} />
        ))}
      </View>
      <Text className="min-w-0 flex-1 font-medium" numberOfLines={1}>
        {ticket.title}
      </Text>
      <Text variant="small" className="shrink-0 font-mono text-muted-foreground">
        {ticketKey(ticket, projectPrefix)}
      </Text>
    </Pressable>
  );
};
