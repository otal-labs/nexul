import { View } from "react-native";

import { TicketCard } from "@/components/board/TicketCard";
import { Text } from "@/components/ui/text";
import type { TicketType } from "@/hooks/TicketTypeHooks";
import { cn } from "@/lib/utils";
import { statusStageDot, type BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

interface StatusSectionProps {
  status: BoardStatus;
  tickets: Ticket[];
  projectPrefix: string | undefined;
  ticketTypes: TicketType[] | undefined;
  onOpenTicket: (id: string) => void;
}

// One status column as a well of cards, in the project's status order; an empty column is a dashed slot.
export const StatusSection = ({ status, tickets, projectPrefix, ticketTypes, onOpenTicket }: StatusSectionProps) => (
  <View className="mx-3 mb-3 gap-2 rounded-xl bg-surface-2 p-2">
    <View className="flex-row items-center gap-2 px-2 pb-0.5 pt-1.5">
      <View className={cn("size-2 rounded-full", statusStageDot(status.kind))} />
      <Text className="flex-1 text-[13px] font-medium">{status.name}</Text>
      <Text className="font-mono text-xs text-muted-foreground">{tickets.length}</Text>
    </View>
    {tickets.length === 0 && (
      <View className="items-center rounded-lg border border-dashed border-input py-4">
        <Text className="text-[13px] text-muted-foreground">No tickets</Text>
      </View>
    )}
    {tickets
      .slice()
      .sort((a, b) => a.position - b.position)
      .map((ticket) => {
        const type = ticketTypes?.find((t) => t.id === ticket.type_id);
        return (
          <TicketCard
            key={ticket.id}
            ticket={ticket}
            stage={status.kind}
            projectPrefix={projectPrefix}
            typeName={type?.name}
            typeColor={type?.color}
            onPress={() => onOpenTicket(ticket.id)}
          />
        );
      })}
  </View>
);
