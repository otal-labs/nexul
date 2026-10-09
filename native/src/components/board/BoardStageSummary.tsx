import { View } from "react-native";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { STATUS_STAGES, StatusKind, type BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

// Backlog is work not yet started, so its share reads quieter than the stages in motion.
const stageTone = (kind: StatusKind, dot: string) => cn(dot, kind === StatusKind.Backlog && "opacity-50");

interface BoardStageSummaryProps {
  statuses: BoardStatus[];
  tickets: Ticket[];
}

// Where the project's work stands: one slim bar split by stage over a legend, every ticket counted whatever the filter shows.
export const BoardStageSummary = ({ statuses, tickets }: BoardStageSummaryProps) => {
  const kindOf = new Map(statuses.map((s) => [s.id, s.kind]));
  const stages = STATUS_STAGES.filter((stage) => statuses.some((s) => s.kind === stage.kind)).map((stage) => ({
    ...stage,
    count: tickets.filter((t) => kindOf.get(t.status) === stage.kind).length,
  }));
  if (tickets.length === 0) return null;
  return (
    <View accessible aria-label="Tickets by stage" className="gap-2.5 px-5 pb-2">
      <View className="h-1.5 flex-row gap-0.5">
        {stages
          .filter((s) => s.count > 0)
          .map((s) => (
            <View key={s.kind} style={{ flexGrow: s.count }} className={cn("h-full rounded-full", stageTone(s.kind, s.dot))} />
          ))}
      </View>
      <View className="flex-row flex-wrap gap-x-4 gap-y-1">
        {stages.map((s) => (
          <View key={s.kind} className="flex-row items-center gap-1.5">
            <View className={cn("size-1.5 rounded-full", stageTone(s.kind, s.dot))} />
            <Text className="text-xs text-muted-foreground">{s.label}</Text>
            <Text className="font-mono text-xs text-foreground">{s.count}</Text>
          </View>
        ))}
      </View>
    </View>
  );
};
