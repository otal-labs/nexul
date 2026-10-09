import { LegendList } from "@legendapp/list/react-native";
import { useMemo, useState } from "react";

import { isContinuation } from "@nexul/client-core/chat";

import { ChatDayDivider } from "@/components/chat/ChatDayDivider";
import { MessageArrival, type Arrival } from "@/components/chat/MessageArrival";
import { MessageRow } from "@/components/chat/MessageRow";
import { useFetchMe } from "@/hooks/AuthHooks";
import { formatDayLabel } from "@/lib/time";
import type { Message } from "@/models/Chat";

interface MessageListProps {
  messages: Message[];
}

type StreamRow =
  | { kind: "day"; key: string; label: string }
  | { kind: "message"; key: string; message: Message; continuation: boolean; own: boolean };

const dayOf = (iso: string) => new Date(iso).toDateString();

// Oldest first and anchored to the end: opens on the newest message and follows new ones while the reader is at the bottom.
export const MessageList = ({ messages }: MessageListProps) => {
  const { data: me } = useFetchMe(true);
  const meId = me?.user.id;
  // What was on screen when the thread opened never animates.
  const [seen] = useState(() => new Set(messages.map((m) => m.id)));

  // Continuation is judged against the full thread, so a deleted message still breaks the run it sat in.
  const rows = useMemo(
    () =>
      messages.flatMap((message, i): StreamRow[] => {
        if (message.deleted_at) return [];
        const prev = messages[i - 1];
        const newDay = !prev || dayOf(prev.created_at) !== dayOf(message.created_at);
        const own = message.author_kind === "user" && message.author_id === meId;
        // The server's copy of a message sent from here keeps the pending row's key, so the row stays mid-rise.
        const key = message.client_key ?? message.id;
        const row: StreamRow = { kind: "message", key, message, continuation: !newDay && isContinuation(prev, message), own };
        if (!newDay) return [row];
        return [{ kind: "day", key: `day-${message.created_at}`, label: formatDayLabel(message.created_at) }, row];
      }),
    [messages, meId],
  );

  // Your confirmed copy replaces the pending row, so only the pending one rises; anyone else's new message arrives.
  const arrivalOf = (message: Message, own: boolean): Arrival | undefined => {
    if (seen.has(message.id)) return undefined;
    if (message.pending) return "rise";
    return own ? undefined : "arrive";
  };

  return (
    <LegendList
      data={rows}
      keyExtractor={(row) => row.key}
      renderItem={({ item }) => (
        <>
          {item.kind === "day" && <ChatDayDivider label={item.label} />}
          {item.kind === "message" && (
            <MessageArrival arrival={arrivalOf(item.message, item.own)}>
              <MessageRow message={item.message} own={item.own} continuation={item.continuation} />
            </MessageArrival>
          )}
        </>
      )}
      estimatedItemSize={72}
      recycleItems={false}
      initialScrollAtEnd
      alignItemsAtEnd
      maintainScrollAtEnd
      maintainVisibleContentPosition
      keyboardShouldPersistTaps="handled"
      contentContainerStyle={{ paddingBottom: 12 }}
    />
  );
};
