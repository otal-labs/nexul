import { LegendList } from "@legendapp/list/react-native";
import { useMemo } from "react";

import { MessageRow } from "@/components/chat/MessageRow";
import { useChatAuthorLookup, useChatWorkspaceId } from "@/hooks/ChatHooks";
import type { Message } from "@/models/Chat";

interface MessageListProps {
  messages: Message[];
}

// Oldest first and anchored to the end: opens on the newest message and follows new ones while the reader is at the bottom.
export const MessageList = ({ messages }: MessageListProps) => {
  const resolveLogin = useChatAuthorLookup(useChatWorkspaceId());
  const visible = useMemo(() => messages.filter((m) => !m.deleted_at), [messages]);
  return (
    <LegendList
      data={visible}
      keyExtractor={(message) => message.id}
      renderItem={({ item }) => <MessageRow message={item} authorLogin={resolveLogin(item.author_id)} />}
      estimatedItemSize={72}
      initialScrollAtEnd
      alignItemsAtEnd
      maintainScrollAtEnd
      maintainVisibleContentPosition
      keyboardShouldPersistTaps="handled"
    />
  );
};
