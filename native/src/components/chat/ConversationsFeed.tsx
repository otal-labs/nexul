import { ScrollView } from "react-native";

import { ConversationSection } from "@/components/chat/ConversationSection";
import { ScreenHeader } from "@/components/ScreenHeader";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchChatUnread } from "@/hooks/ChatHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { groupConversations, type Conversation, type ConversationGroup, type DMLabelContext, type UnreadCounts } from "@/models/Chat";

// Counts only the rows listed here, so a voice channel's unread never shows as a number with no row behind it.
const unreadLine = (groups: ConversationGroup[], unread: UnreadCounts | undefined) => {
  const count = groups.flatMap((g) => g.conversations).filter((c) => (unread?.[c.id] ?? 0) > 0).length;
  return count === 0 ? undefined : `${count} with unread messages`;
};

interface ConversationsFeedProps {
  workspaceId: string | undefined;
  conversations: Conversation[];
}

export const ConversationsFeed = ({ workspaceId, conversations }: ConversationsFeedProps) => {
  const workspace = useSelectedWorkspace();
  const { data: me } = useFetchMe(true);
  const { data: unread } = useFetchChatUnread(workspaceId);
  const resolvePerson = usePersonLookup(workspaceId);
  const groups = groupConversations(conversations);
  const dmCtx: DMLabelContext = { currentUserId: me?.user.id, resolvePerson };

  return (
    <ScrollView className="flex-1" contentContainerClassName="pb-6">
      <ScreenHeader eyebrow={workspace?.name} title="Chat" meta={unreadLine(groups, unread)} />
      {groups.map((group) => (
        <ConversationSection key={group.title} group={group} unread={unread} dmCtx={dmCtx} />
      ))}
    </ScrollView>
  );
};
