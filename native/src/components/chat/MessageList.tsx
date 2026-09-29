import { LegendList } from "@legendapp/list/react-native";
import { useMemo } from "react";

import { MessageRow } from "@/components/chat/MessageRow";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { isContinuation, type Message } from "@/models/Chat";
import { personLabel } from "@/models/Person";

interface MessageListProps {
  messages: Message[];
}

// Oldest first and anchored to the end: opens on the newest message and follows new ones while the reader is at the bottom.
export const MessageList = ({ messages }: MessageListProps) => {
  const resolvePerson = usePersonLookup(useCurrentWorkspaceId());
  // Continuation is judged against the full thread, so a deleted message still breaks the run it sat in.
  const rows = useMemo(
    () => messages.flatMap((message, i) => (message.deleted_at ? [] : [{ message, continuation: isContinuation(messages[i - 1], message) }])),
    [messages],
  );
  return (
    <LegendList
      data={rows}
      keyExtractor={(row) => row.message.id}
      renderItem={({ item }) => (
        <MessageRow message={item.message} authorName={personLabel(resolvePerson(item.message.author_id))} continuation={item.continuation} />
      )}
      estimatedItemSize={72}
      recycleItems={false}
      initialScrollAtEnd
      alignItemsAtEnd
      maintainScrollAtEnd
      maintainVisibleContentPosition
      keyboardShouldPersistTaps="handled"
    />
  );
};
