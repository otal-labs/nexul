import { memo, useContext } from "react";
import { Pressable, View } from "react-native";
import { GestureDetector } from "react-native-gesture-handler";
import Animated from "react-native-reanimated";

import { BoardDragContext, useCardDrag, useLifted } from "@/components/board/boardDrag";
import { BoardLabelColorsContext } from "@/components/board/boardLabelColors";
import { labelDot, labelDotStyle } from "@/components/board/ticketTypeColor";
import { TicketTypeBadge } from "@/components/board/TicketTypeBadge";
import { PersonAvatar } from "@/components/PersonAvatar";
import { Text } from "@/components/ui/text";
import { useTicketPersonLookup } from "@/hooks/PeopleHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { cn } from "@/lib/utils";
import type { StatusKind } from "@/models/Status";
import { cardPerson, ticketKey, type Ticket } from "@/models/Ticket";

export interface TicketRowBodyProps {
  ticket: Ticket;
  stage: StatusKind;
  projectPrefix: string | undefined;
  typeName: string | undefined;
  typeColor: string | undefined;
}

export const ticketRowClass = "min-h-14 flex-row items-start gap-3 px-5 py-2.5";

// The title over one meta line (the mono key, the type, each label as a dot and its name), whoever acts next trailing.
export const TicketRowBody = ({ ticket, stage, projectPrefix, typeName, typeColor }: TicketRowBodyProps) => {
  const resolvePerson = useTicketPersonLookup(useCurrentWorkspaceId());
  const labelColors = useContext(BoardLabelColorsContext);
  const person = cardPerson(ticket, stage);
  const labels = ticket.labels ?? [];
  return (
    <>
      <View className="min-w-0 flex-1 gap-1">
        <Text numberOfLines={2} className="text-body font-medium">
          {ticket.title}
        </Text>
        <View className="flex-row flex-wrap items-center gap-x-3 gap-y-1">
          <Text className="font-mono text-xs text-muted-foreground">{ticketKey(ticket, projectPrefix)}</Text>
          {typeName && <TicketTypeBadge name={typeName} color={typeColor} />}
          {labels.map((label) => (
            <View key={label} className="flex-row items-center gap-1">
              <View className={cn("size-1.5 rounded-full", labelDot(label, labelColors?.[label]))} style={labelDotStyle(labelColors?.[label])} />
              <Text className="text-xs text-muted-foreground">{label}</Text>
            </View>
          ))}
        </View>
      </View>
      {person.login && <PersonAvatar person={resolvePerson(person)} size={24} />}
    </>
  );
};

interface TicketRowProps extends TicketRowBodyProps {
  onOpen: (id: string) => void;
}

// Tap opens the ticket; a long press picks the row up to drop on another status. The same moves are accessibility actions.
// Memoized: a frame that changes one ticket keeps the others' references, so only that row renders again.
const TicketRowView = ({ onOpen, ...body }: TicketRowProps) => {
  const drag = useContext(BoardDragContext);
  const { ref, gesture } = useCardDrag(body.ticket);
  const lifted = useLifted(drag, (held) => held?.ticket.id === body.ticket.id);
  const moves = (drag?.statuses ?? []).filter((s) => s.id !== body.ticket.status);
  const row = (
    <Animated.View ref={ref} collapsable={false} style={{ opacity: lifted ? 0.35 : 1 }}>
      <Pressable
        role="button"
        aria-label={`${ticketKey(body.ticket, body.projectPrefix)} ${body.ticket.title}`}
        accessibilityHint={drag ? "Hold to move it to another status" : undefined}
        accessibilityActions={moves.map((s) => ({ name: s.id, label: `Move to ${s.name}` }))}
        onAccessibilityAction={({ nativeEvent }) => drag?.move(body.ticket.id, nativeEvent.actionName)}
        onPress={() => onOpen(body.ticket.id)}
        className={cn(ticketRowClass, "active:bg-accent")}
      >
        <TicketRowBody {...body} />
      </Pressable>
    </Animated.View>
  );
  return (
    <>
      {gesture && <GestureDetector gesture={gesture}>{row}</GestureDetector>}
      {!gesture && row}
    </>
  );
};

export const TicketRow = memo(TicketRowView);
