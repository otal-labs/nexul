import { LegendList } from "@legendapp/list/react-native";
import { useMemo } from "react";

import { MessageRow } from "@/components/chat/MessageRow";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import type { Message } from "@/models/Chat";
import { personLabel } from "@/models/Person";

interface MessageListProps {
  messages: Message[];
}

// Oldest first and anchored to the end: opens on the newest message and follows new ones while the reader is at the bottom.
export const MessageList = ({ messages }: MessageListProps) => {
  const resolvePerson = usePersonLookup(useCurrentWorkspaceId());
  const visible = useMemo(() => messages.filter((m) => !m.deleted_at), [messages]);
  return (
    <LegendList
      data={visible}
      keyExtractor={(message) => message.id}
      renderItem={({ item }) => <MessageRow message={item} authorName={personLabel(resolvePerson(item.author_id))} />}
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
