import { FileText } from "lucide-react";

import { RowActions } from "@/components/listpane/RowActions";
import { ChatSidebarRow } from "@/components/sidebar/ChatSidebarRow";
import { useFetchDoc } from "@/hooks/DocHooks";
import { useHiddenThreads } from "@/hooks/useHiddenThreads";
import type { Conversation } from "@/models/Chat";

interface DocThreadSidebarRowProps {
  conversation: Conversation;
  unreadCount: number;
}

// Titled with the doc, not the thread; the Threads heading already says what it is.
export const DocThreadSidebarRow = ({ conversation, unreadCount }: DocThreadSidebarRowProps) => {
  const { data: doc } = useFetchDoc(conversation.doc_id);
  const { hiddenIds, toggle } = useHiddenThreads();
  const label = doc?.title ?? "Doc thread";
  return (
    <ChatSidebarRow
      conversationId={conversation.id}
      label={label}
      icon={FileText}
      unreadCount={unreadCount}
      actions={<RowActions itemLabel={label} sidebar={{ hidden: hiddenIds.includes(conversation.id), onToggle: () => toggle(conversation.id) }} />}
    />
  );
};
