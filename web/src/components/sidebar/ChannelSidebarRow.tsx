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
  const { onSettings, onRename, onDelete, settingsDialog } = useChannelRowActions(conversation);
  return (
    <>
      <ChatSidebarRow
        conversationId={conversation.id}
        label={conversation.name ?? "Channel"}
        icon={Hash}
        isPrivate={conversation.private}
        unreadCount={unreadCount}
        actions={
          (onSettings || onRename || onDelete) && (
            <RowActions itemLabel={channelMention(conversation)} onSettings={onSettings} onRename={onRename} onDelete={onDelete} />
          )
        }
      />
      {settingsDialog}
    </>
  );
};
