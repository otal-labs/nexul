import { View } from "react-native";

import { Text } from "@/components/ui/text";
import { TicketRow } from "@/components/board/TicketRow";
import type { BoardStatus } from "@/models/Status";
import { statusStageDot } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";
import type { TicketType } from "@/hooks/TicketTypeHooks";
import { cn } from "@/lib/utils";

interface StatusSectionProps {
  status: BoardStatus;
  tickets: Ticket[];
  projectPrefix: string | undefined;
  ticketTypes: TicketType[] | undefined;
  onOpenTicket: (id: string) => void;
}

// One status column rendered as a section, in the project's status order (ticket 28).
export const StatusSection = ({ status, tickets, projectPrefix, ticketTypes, onOpenTicket }: StatusSectionProps) => (
  <View>
    <View className="flex-row items-center gap-2 bg-surface-2 px-4 py-2">
      <View className={cn("size-2 rounded-full", statusStageDot(status.kind))} />
      <Text variant="small" className="flex-1 font-medium">
        {status.name}
      </Text>
      <Text variant="small" className="font-mono text-muted-foreground">
        {tickets.length}
      </Text>
    </View>
    {tickets.length === 0 && (
      <Text variant="muted" className="px-4 py-3">
        No tickets
      </Text>
    )}
    {tickets
      .slice()
      .sort((a, b) => a.position - b.position)
      .map((ticket) => {
        const type = ticketTypes?.find((t) => t.id === ticket.type_id);
        return (
          <TicketRow
            key={ticket.id}
            ticket={ticket}
            projectPrefix={projectPrefix}
            typeName={type?.name}
            typeColor={type?.color}
            onPress={() => onOpenTicket(ticket.id)}
          />
        );
      })}
  </View>
);
