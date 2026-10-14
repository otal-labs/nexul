import { Hash } from "lucide-react";

import type { Conversation } from "@nexul/client-core/chat";

import { ChatSidebarRow } from "@/components/sidebar/ChatSidebarRow";
import { useChannelRowActions } from "@/hooks/useChannelRowActions";

interface ChannelSidebarRowProps {
  conversation: Conversation;
  unreadCount: number;
}

export const ChannelSidebarRow = ({ conversation, unreadCount }: ChannelSidebarRowProps) => {
  const { gates, deleteGate, onSettings, onRename, permissions, onDelete, settingsDialog } = useChannelRowActions(conversation);
  return (
    <>
      <ChatSidebarRow
        conversationId={conversation.id}
        label={conversation.name ?? "Channel"}
        icon={Hash}
        isPrivate={conversation.private}
        unreadCount={unreadCount}
        menu={{ gates, deleteGate, onSettings, onRename, permissions, onDelete }}
      />
      {settingsDialog}
    </>
  );
};
