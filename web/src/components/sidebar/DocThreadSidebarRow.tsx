import { FileText } from "lucide-react";

import { ChatSidebarRow } from "@/components/sidebar/ChatSidebarRow";
import { useFetchDoc } from "@/hooks/DocHooks";
import type { Conversation } from "@/models/Chat";

interface DocThreadSidebarRowProps {
  conversation: Conversation;
  unreadCount: number;
}

// Titled with the doc, not the thread; the Threads heading already says what it is.
export const DocThreadSidebarRow = ({ conversation, unreadCount }: DocThreadSidebarRowProps) => {
  const { data: doc } = useFetchDoc(conversation.doc_id);
  return (
    <ChatSidebarRow conversationId={conversation.id} label={doc?.title ?? "Doc thread"} icon={FileText} unreadCount={unreadCount} />
  );
};
