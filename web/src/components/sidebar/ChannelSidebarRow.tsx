import { Hash } from "lucide-react";

import { RowActions } from "@/components/listpane/RowActions";
import { ChatSidebarRow } from "@/components/sidebar/ChatSidebarRow";
import { useChannelRowActions } from "@/hooks/useChannelRowActions";
import { channelMention, type Conversation } from "@/models/Chat";

interface ChannelSidebarRowProps {
  conversation: Conversation;
  unreadCount: number;
}

export const ChannelSidebarRow = ({ conversation, unreadCount }: ChannelSidebarRowProps) => {
  const { onRename, onDelete } = useChannelRowActions(conversation);
  return (
    <ChatSidebarRow
      conversationId={conversation.id}
      label={conversation.name ?? "Channel"}
      icon={Hash}
      unreadCount={unreadCount}
      actions={(onRename || onDelete) && <RowActions itemLabel={channelMention(conversation)} onRename={onRename} onDelete={onDelete} />}
    />
  );
};
