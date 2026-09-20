import { FileText } from "lucide-react";

import { conversationRowClass, UnreadDot } from "@/components/chat/ConversationRow";
import { useFetchDoc } from "@/hooks/DocHooks";
import { cn } from "@/lib/utils";
import type { Conversation } from "@/models/Chat";

interface DocThreadRowProps {
  conversation: Conversation;
  unreadCount: number;
  selected: boolean;
  onSelect: (conversationId: string) => void;
}

// The row is titled with the doc, not the thread; the Threads group already says what it is.
export const DocThreadRow = ({ conversation, unreadCount, selected, onSelect }: DocThreadRowProps) => {
  const { data: doc } = useFetchDoc(conversation.doc_id);
  return (
    <button
      type="button"
      onClick={() => onSelect(conversation.id)}
      aria-current={selected ? "true" : undefined}
      className={conversationRowClass(selected)}
    >
      <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <FileText className="size-3.5" aria-hidden />
      </span>
      <span className={cn("min-w-0 flex-1 truncate", unreadCount > 0 && "font-medium")}>{doc?.title ?? "Doc thread"}</span>
      {unreadCount > 0 && <UnreadDot count={unreadCount} />}
    </button>
  );
};
