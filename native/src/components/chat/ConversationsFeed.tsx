import { ScrollView } from "react-native";

import { ConversationSection } from "@/components/chat/ConversationSection";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useChatAuthorLookup, useFetchChatUnread } from "@/hooks/ChatHooks";
import { groupConversations, type Conversation, type DMLabelContext } from "@/models/Chat";

interface ConversationsFeedProps {
  workspaceId: string | undefined;
  conversations: Conversation[];
}

export const ConversationsFeed = ({ workspaceId, conversations }: ConversationsFeedProps) => {
  const { data: me } = useFetchMe(true);
  const { data: unread } = useFetchChatUnread(workspaceId);
  const resolveLogin = useChatAuthorLookup(workspaceId);
  const dmCtx: DMLabelContext = { currentUserId: me?.user.id, resolveLogin };

  return (
    <ScrollView className="flex-1" contentContainerClassName="pb-4">
      {groupConversations(conversations).map((group) => (
        <ConversationSection key={group.title} group={group} unread={unread} dmCtx={dmCtx} />
      ))}
    </ScrollView>
  );
};
