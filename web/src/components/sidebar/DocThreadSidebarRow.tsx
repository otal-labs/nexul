import { FileText } from "lucide-react";

import type { Conversation } from "@nexul/client-core/chat";

import { ChatSidebarRow } from "@/components/sidebar/ChatSidebarRow";
import { useFetchDoc } from "@/hooks/DocHooks";
import { useBotsDialog } from "@/hooks/useBotsDialog";
import { useHiddenThreads } from "@/hooks/useHiddenThreads";

interface DocThreadSidebarRowProps {
  conversation: Conversation;
  unreadCount: number;
}

// Titled with the doc, not the thread; the Threads heading already says what it is.
export const DocThreadSidebarRow = ({ conversation, unreadCount }: DocThreadSidebarRowProps) => {
  const { data: doc } = useFetchDoc(conversation.doc_id);
  const { hiddenIds, toggle } = useHiddenThreads();
  const label = doc?.title ?? "Doc thread";
  const { onBots, botsDialog } = useBotsDialog(conversation, label);
  return (
    <>
      <ChatSidebarRow
        conversationId={conversation.id}
        label={label}
        icon={FileText}
        unreadCount={unreadCount}
        menu={{ onBots, sidebar: { hidden: hiddenIds.includes(conversation.id), onToggle: () => toggle(conversation.id) } }}
      />
      {botsDialog}
    </>
  );
};
