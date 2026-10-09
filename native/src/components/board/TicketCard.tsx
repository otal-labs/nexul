import { useContext } from "react";
import { Pressable, View } from "react-native";
import { GestureDetector } from "react-native-gesture-handler";
import Animated from "react-native-reanimated";

import { BoardDragContext, useCardDrag, useLifted } from "@/components/board/boardDrag";
import { labelPill, ticketTypePill } from "@/components/board/ticketTypeColor";
import { PersonAvatar } from "@/components/PersonAvatar";
import { Text } from "@/components/ui/text";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { cn } from "@/lib/utils";
import type { StatusKind } from "@/models/Status";
import { cardPerson, ticketKey, type Ticket } from "@/models/Ticket";

export interface TicketCardBodyProps {
  ticket: Ticket;
  stage: StatusKind;
  projectPrefix: string | undefined;
  typeName: string | undefined;
  typeColor: string | undefined;
}

const Pill = ({ className, label }: { className: string; label: string }) => (
  <Text numberOfLines={1} className={cn("overflow-hidden rounded-full px-2 py-0.5 text-xs font-medium", className)}>
    {label}
  </Text>
);

// The mono key as an eyebrow, the title at full width, then the type and label pills with whoever acts next opposite.
export const TicketCardBody = ({ ticket, stage, projectPrefix, typeName, typeColor }: TicketCardBodyProps) => {
  const resolvePerson = usePersonLookup(useCurrentWorkspaceId());
  const person = cardPerson(ticket, stage);
  const labels = ticket.labels ?? [];
  return (
    <>
      <Text className="font-mono text-[11px] text-muted-foreground">{ticketKey(ticket, projectPrefix)}</Text>
      <Text numberOfLines={3} className="text-[15px] font-medium leading-5">
        {ticket.title}
      </Text>
      {(typeName || labels.length > 0 || person.login) && (
        <View className="flex-row items-end gap-2">
          <View className="min-w-0 flex-1 flex-row flex-wrap gap-1.5">
            {typeName && <Pill className={ticketTypePill(typeName, typeColor)} label={typeName} />}
            {labels.map((label) => (
              <Pill key={label} className={labelPill(label)} label={label} />
            ))}
          </View>
          {person.login && <PersonAvatar person={resolvePerson(person.login)} size={22} />}
        </View>
      )}
    </>
  );
};

export const ticketCardClass = "gap-2 rounded-lg border border-border bg-card px-3.5 py-3";

interface TicketCardProps extends TicketCardBodyProps {
  onPress: () => void;
}

// Tap opens the ticket; a long press picks the card up to drop on another column. The same moves are accessibility actions.
export const TicketCard = ({ onPress, ...body }: TicketCardProps) => {
  const drag = useContext(BoardDragContext);
  const { ref, gesture } = useCardDrag(body.ticket);
  const lifted = useLifted(drag, (held) => held?.ticket.id === body.ticket.id);
  const moves = (drag?.statuses ?? []).filter((s) => s.id !== body.ticket.status);
  const card = (
    <Animated.View ref={ref} collapsable={false} style={{ opacity: lifted ? 0.35 : 1 }}>
      <Pressable
        role="button"
        aria-label={`${ticketKey(body.ticket, body.projectPrefix)} ${body.ticket.title}`}
        accessibilityHint="Hold to move it to another column"
        accessibilityActions={moves.map((s) => ({ name: s.id, label: `Move to ${s.name}` }))}
        onAccessibilityAction={({ nativeEvent }) => drag?.move(body.ticket.id, nativeEvent.actionName)}
        onPress={onPress}
        className={cn(ticketCardClass, "active:bg-accent")}
      >
        <TicketCardBody {...body} />
      </Pressable>
    </Animated.View>
  );
  return (
    <>
      {gesture && <GestureDetector gesture={gesture}>{card}</GestureDetector>}
      {!gesture && card}
    </>
  );
};
