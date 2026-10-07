import { Users } from "lucide-react";

import { RowActions } from "@/components/listpane/RowActions";
import { ChatSidebarRow } from "@/components/sidebar/ChatSidebarRow";
import { useBotsDialog } from "@/hooks/useBotsDialog";
import type { Conversation } from "@/models/Chat";

interface DMSidebarRowProps {
  conversation: Conversation;
  label: string;
  unreadCount: number;
}

export const DMSidebarRow = ({ conversation, label, unreadCount }: DMSidebarRowProps) => {
  const { onBots, botsDialog } = useBotsDialog(conversation, label);
  return (
    <>
      <ChatSidebarRow
        conversationId={conversation.id}
        label={label}
        icon={Users}
        unreadCount={unreadCount}
        actions={onBots && <RowActions itemLabel={label} onBots={onBots} />}
      />
      {botsDialog}
    </>
  );
};
