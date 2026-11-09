import { View } from "react-native";

import { TicketCard } from "@/components/board/TicketCard";
import { Text } from "@/components/ui/text";
import type { TicketType } from "@/hooks/TicketTypeHooks";
import { cn } from "@/lib/utils";
import { statusStageDot, type BoardStatus, type StatusKind } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

interface RowBase {
  key: string;
  // The last row of a column closes its well and leaves the gap before the next one.
  last: boolean;
}

export type BoardRow =
  | (RowBase & { kind: "head"; status: BoardStatus; count: number })
  | (RowBase & { kind: "empty" })
  | (RowBase & { kind: "card"; ticket: Ticket; stage: StatusKind; typeName: string | undefined; typeColor: string | undefined });

// Each column is flattened to a head row, then its cards in position order (or an empty slot), so a list can mount only the rows in view.
export const buildBoardRows = (statuses: BoardStatus[], tickets: Ticket[], ticketTypes: TicketType[] | undefined): BoardRow[] => {
  const types = new Map(ticketTypes?.map((type) => [type.id, type]));
  const byStatus = new Map<string, Ticket[]>();
  for (const ticket of tickets) byStatus.set(ticket.status, [...(byStatus.get(ticket.status) ?? []), ticket]);
  return statuses.flatMap((status): BoardRow[] => {
    const column = (byStatus.get(status.id) ?? []).sort((a, b) => a.position - b.position);
    const head: BoardRow = { kind: "head", key: `head-${status.id}`, status, count: column.length, last: column.length === 0 };
    if (column.length === 0) return [head, { kind: "empty", key: `empty-${status.id}`, last: true }];
    return [
      head,
      ...column.map((ticket, i): BoardRow => {
        const type = types.get(ticket.type_id);
        return { kind: "card", key: ticket.id, ticket, stage: status.kind, typeName: type?.name, typeColor: type?.color, last: i === column.length - 1 };
      }),
    ];
  });
};

interface BoardRowViewProps {
  row: BoardRow;
  projectPrefix: string | undefined;
  onOpenTicket: (id: string) => void;
}

// One row of a status column's well: the rows stack as one rounded panel, in the project's status order.
export const BoardRowView = ({ row, projectPrefix, onOpenTicket }: BoardRowViewProps) => (
  <View className={cn("px-3", row.last && "pb-3")}>
    <View className={cn("bg-surface-2 px-2 pt-2", row.kind === "head" && "rounded-t-xl", row.last && "rounded-b-xl pb-2")}>
      {row.kind === "head" && (
        <View className="flex-row items-center gap-2 px-2 pb-0.5 pt-1.5">
          <View className={cn("size-2 rounded-full", statusStageDot(row.status.kind))} />
          <Text className="flex-1 text-[13px] font-medium">{row.status.name}</Text>
          <Text className="font-mono text-xs text-muted-foreground">{row.count}</Text>
        </View>
      )}
      {row.kind === "empty" && (
        <View className="items-center rounded-lg border border-dashed border-input py-4">
          <Text className="text-[13px] text-muted-foreground">No tickets</Text>
        </View>
      )}
      {row.kind === "card" && (
        <TicketCard
          ticket={row.ticket}
          stage={row.stage}
          projectPrefix={projectPrefix}
          typeName={row.typeName}
          typeColor={row.typeColor}
          onOpen={onOpenTicket}
        />
      )}
    </View>
  </View>
);
