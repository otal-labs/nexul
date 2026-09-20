import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { ChatQuestionCard } from "@/components/chat/ChatQuestionCard";
import { MessageBody } from "@/components/chat/MessageBody";
import { MessageEditForm } from "@/components/chat/MessageEditForm";
import { MessageRowAvatar, MessageRowHeader, type MessageAlign } from "@/components/chat/MessageRowHeader";
import { MessageTrailTurns } from "@/components/chat/MessageTrailTurns";
import { TrailQuestionBody } from "@/components/play/TrailQuestionCard";
import { TrailReplyProse } from "@/components/play/TrailReplyProse";
import { Button } from "@/components/ui/button";
import { Message, MessageAvatar, MessageContent } from "@/components/ui/message";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { cn } from "@/lib/utils";
import type { Message as ChatMessage } from "@/models/Chat";
import { parseQuestionMessage } from "@/models/Question";
import type { TrailBlock } from "@/utils/ThreadTrailUtility";

interface MessageRowProps {
  message: ChatMessage;
  authorLogin: string;
  isOwn: boolean;
  // questionAnswered says the thread already moved past an Agent question, so its card is read-only.
  questionAnswered?: boolean;
  // trailBlock is the run that led to this message: its turns render above it, and its question is answered on the trail.
  trailBlock?: TrailBlock | undefined;
  onEdit: (messageId: string, body: string) => Promise<void>;
  onDelete: (messageId: string) => Promise<void>;
}

const MessageBubble = ({ message, align }: { message: ChatMessage; align: MessageAlign }) => (
  <div
    data-slot="bubble"
    className={cn(
      "max-w-[85%] space-y-1.5 rounded-2xl bg-accent px-3 py-2 text-sm break-words text-accent-foreground",
      align === "end" ? "rounded-br-md" : "rounded-bl-md",
    )}
  >
    <MessageBody body={message.body} mentionHandles={(message.mentions ?? []).map((m) => m.handle)} />
  </div>
);

const SystemMessageRow = ({ body, trailBlock }: { body: string; trailBlock: TrailBlock | undefined }) => (
  <div className="px-3 py-1">
    {trailBlock && <MessageTrailTurns turns={trailBlock.turns} />}
    <p className="text-xs text-muted-foreground italic">{body}</p>
  </div>
);

interface AgentMessageBodyProps {
  message: ChatMessage;
  trailBlock: TrailBlock | undefined;
  questionAnswered: boolean;
}

// The Agent's message in the trail dialog's shape: its turns as collapsible groups, then the reply as prose or the
// question card where it asked.
const AgentMessageBody = ({ message, trailBlock, questionAnswered }: AgentMessageBodyProps) => {
  const question = parseQuestionMessage(message.body);
  return (
    <>
      {trailBlock && trailBlock.turns.length > 0 && <MessageTrailTurns turns={trailBlock.turns} />}
      {question === null && <TrailReplyProse text={message.body} />}
      {question !== null && trailBlock && <TrailQuestionBody trail={trailBlock.trail} />}
      {question !== null && !trailBlock && (
        <ChatQuestionCard conversationId={message.conversation_id} question={question} answered={questionAnswered} />
      )}
    </>
  );
};

// Your own messages sit right-aligned with no header; everyone else gets an avatar + name/time header.
export const MessageRow = ({ message, authorLogin, isOwn, questionAnswered = false, trailBlock, onEdit, onDelete }: MessageRowProps) => {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(message.body);
  const { open: confirmDelete } = useConfirmationDialog();

  if (message.deleted_at) return null;

  const isSystem = message.author_kind === "system";
  const isAgent = message.author_kind === "agent";
  const canEditOrDelete = isOwn && !isAgent;
  const align: MessageAlign = canEditOrDelete ? "end" : "start";

  const startEdit = () => {
    setDraft(message.body);
    setEditing(true);
  };

  const saveEdit = async () => {
    const trimmed = draft.trim();
    if (trimmed === "" || trimmed === message.body) {
      setEditing(false);
      return;
    }
    await onEdit(message.id, trimmed);
    setEditing(false);
  };

  const remove = async () => {
    const confirmed = await confirmDelete({ message: "Delete this message?" });
    if (confirmed) await onDelete(message.id);
  };

  return (
    <>
      {isSystem && <SystemMessageRow body={message.body} trailBlock={trailBlock} />}
      {!isSystem && (
        <Message align={align} className={cn("group px-3 py-1", message.pending && "opacity-60")}>
          {align === "start" && (
            <MessageAvatar className="size-6 self-start bg-transparent">
              <MessageRowAvatar isAgent={isAgent} authorLogin={authorLogin} />
            </MessageAvatar>
          )}
          <MessageContent>
            <MessageRowHeader align={align} message={message} isAgent={isAgent} authorLogin={authorLogin} />
            {editing && (
              <MessageEditForm draft={draft} onDraftChange={setDraft} onCancel={() => setEditing(false)} onSave={() => void saveEdit()} />
            )}
            {!editing && !isAgent && <MessageBubble message={message} align={align} />}
            {!editing && isAgent && <AgentMessageBody message={message} trailBlock={trailBlock} questionAnswered={questionAnswered} />}
          </MessageContent>
          {canEditOrDelete && !editing && (
            <div className="flex h-fit shrink-0 gap-0.5 self-center opacity-0 transition-opacity duration-150 ease-standard group-hover:opacity-100">
              <Button size="icon" variant="ghost" className="size-6" aria-label="Edit message" onClick={startEdit}>
                <Pencil className="size-3.5" aria-hidden />
              </Button>
              <Button size="icon" variant="ghost" className="size-6" aria-label="Delete message" onClick={() => void remove()}>
                <Trash2 className="size-3.5" aria-hidden />
              </Button>
            </div>
          )}
        </Message>
      )}
    </>
  );
};
