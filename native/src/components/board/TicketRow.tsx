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

const MAX_LABELS = 2;

const DotName = ({ dotClass, name }: { dotClass: string; name: string }) => (
  <View className="shrink flex-row items-center gap-1">
    <View className={cn("size-2 shrink-0 rounded-full", dotClass)} />
    <Text variant="muted" numberOfLines={1} className="shrink text-xs">
      {name}
    </Text>
  </View>
);

// Hairline row per practices/native.md: title left, mono ticket key right; type and labels sit as dot plus name under the title.
export const TicketRow = ({ ticket, projectPrefix, typeName, typeColor, onPress }: TicketRowProps) => {
  const labels = ticket.labels ?? [];
  const hiddenLabels = labels.length - MAX_LABELS;
  return (
    <Pressable
      role="button"
      onPress={onPress}
      className="min-h-11 gap-1 border-b border-border px-4 py-2.5 active:bg-accent"
    >
      <View className="flex-row items-center gap-2">
        <Text className="min-w-0 flex-1 font-medium" numberOfLines={1}>
          {ticket.title}
        </Text>
        <Text variant="small" className="shrink-0 font-mono text-muted-foreground">
          {ticketKey(ticket, projectPrefix)}
        </Text>
      </View>
      {(typeName || labels.length > 0) && (
        <View className="flex-row items-center gap-3">
          {typeName && <DotName dotClass={ticketTypeDotColor(typeName, typeColor)} name={typeName} />}
          {labels.slice(0, MAX_LABELS).map((label) => (
            <DotName key={label} dotClass={labelDotColor(label)} name={label} />
          ))}
          {hiddenLabels > 0 && (
            <Text variant="muted" className="shrink-0 text-xs">
              +{hiddenLabels}
            </Text>
          )}
        </View>
      )}
    </Pressable>
  );
};
