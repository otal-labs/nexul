import { View } from "react-native";

import { ConversationRow } from "@/components/chat/ConversationRow";
import { Text } from "@/components/ui/text";
import type { ConversationGroup, DMLabelContext, UnreadCounts } from "@/models/Chat";

interface ConversationSectionProps {
  group: ConversationGroup;
  unread: UnreadCounts | undefined;
  dmCtx: DMLabelContext;
}

export const ConversationSection = ({ group, unread, dmCtx }: ConversationSectionProps) => (
  <View>
    <Text className="px-4 pb-1 pt-4 text-xs font-medium uppercase tracking-wide text-muted-foreground">
      {group.title}
    </Text>
    {group.conversations.map((conversation) => (
      <ConversationRow
        key={conversation.id}
        conversation={conversation}
        unreadCount={unread?.[conversation.id] ?? 0}
        dmCtx={dmCtx}
      />
    ))}
  </View>
);
