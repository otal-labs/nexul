import { View } from "react-native";

import { ConversationRow } from "@/components/chat/ConversationRow";
import { Microheader } from "@/components/Microheader";
import type { ConversationGroup, DMLabelContext, UnreadCounts } from "@/models/Chat";

interface ConversationSectionProps {
  group: ConversationGroup;
  unread: UnreadCounts | undefined;
  dmCtx: DMLabelContext;
}

export const ConversationSection = ({ group, unread, dmCtx }: ConversationSectionProps) => (
  <View className="pb-2">
    <Microheader className="px-5 pb-1.5 pt-4">{group.title}</Microheader>
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
