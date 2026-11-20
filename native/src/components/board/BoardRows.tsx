import ChevronDown from "lucide-react-native/icons/chevron-down";
import ChevronRight from "lucide-react-native/icons/chevron-right";
import { memo } from "react";
import { Pressable, View } from "react-native";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import type { TicketType } from "@/hooks/TicketTypeHooks";
import { cn } from "@/lib/utils";
import { StatusKind, statusStageDot, type BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

export type BoardRow =
  | { kind: "head"; key: string; status: BoardStatus; count: number; open: boolean }
  | { kind: "ticket"; key: string; ticket: Ticket; stage: StatusKind; typeName: string | undefined; typeColor: string | undefined };

// A status the viewer never folded or opened starts folded when it is a done status, since finished work is counted, not read.
export const isFolded = (status: BoardStatus, folds: Record<string, boolean> | undefined) => folds?.[status.id] ?? status.kind === StatusKind.Done;

// Each status is flattened to a head row, then its tickets in position order unless it is folded, so a list mounts only the rows in view.
export const buildBoardRows = (
  statuses: BoardStatus[],
  tickets: Ticket[],
  ticketTypes: TicketType[] | undefined,
  folds: Record<string, boolean> | undefined,
): BoardRow[] => {
  const types = new Map(ticketTypes?.map((type) => [type.id, type]));
  const byStatus = new Map<string, Ticket[]>();
  for (const ticket of tickets) {
    const column = byStatus.get(ticket.status);
    if (column) column.push(ticket);
    if (!column) byStatus.set(ticket.status, [ticket]);
  }
  return statuses.flatMap((status): BoardRow[] => {
    const column = (byStatus.get(status.id) ?? []).sort((a, b) => a.position - b.position);
    const open = !isFolded(status, folds);
    const head: BoardRow = { kind: "head", key: `head-${status.id}`, status, count: column.length, open };
    if (!open) return [head];
    return [
      head,
      ...column.map((ticket): BoardRow => {
        const type = types.get(ticket.type_id);
        return { kind: "ticket", key: ticket.id, ticket, stage: status.kind, typeName: type?.name, typeColor: type?.color };
      }),
    ];
  });
};

interface StatusHeadRowProps {
  status: BoardStatus;
  count: number;
  open: boolean;
  onToggle: (statusId: string, open: boolean) => void;
}

// A status's section head: its stage dot, name and mono count; a tap folds or opens the tickets under it.
const StatusHeadRowView = ({ status, count, open, onToggle }: StatusHeadRowProps) => (
  <Pressable
    role="button"
    aria-label={`${status.name}, ${count} ${count === 1 ? "ticket" : "tickets"}`}
    aria-expanded={open}
    onPress={() => onToggle(status.id, open)}
    className="mt-3 min-h-11 flex-row items-center gap-2 px-5 active:bg-accent"
  >
    <View className={cn("size-2 rounded-full", statusStageDot(status.kind))} />
    <Text className="flex-1 text-row font-semibold">{status.name}</Text>
    <Text className="font-mono text-xs text-muted-foreground">{count}</Text>
    <Icon as={open ? ChevronDown : ChevronRight} className="size-4 text-muted-foreground" />
  </Pressable>
);

export const StatusHeadRow = memo(StatusHeadRowView);
