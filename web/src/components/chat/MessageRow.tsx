import { useState, type ReactNode } from "react";

import { BotMessageRow } from "@/components/chat/BotMessageRow";
import { ChatQuestionCard } from "@/components/chat/ChatQuestionCard";
import { MessageActions } from "@/components/chat/MessageActions";
import { MessageBody } from "@/components/chat/MessageBody";
import { MessageEditForm } from "@/components/chat/MessageEditForm";
import { MessageReactions } from "@/components/chat/MessageReactions";
import { MessageContinuationTime, MessageRowAvatar, MessageRowHeader, type MessageAlign } from "@/components/chat/MessageRowHeader";
import { MessageTrailTurns } from "@/components/chat/MessageTrailTurns";
import { HandoffPills } from "@/components/handoff/HandoffPills";
import { NoteMessage } from "@/components/note/NoteMessage";
import { TrailQuestionBody } from "@/components/play/TrailQuestionCard";
import { TrailReplyProse } from "@/components/play/TrailReplyProse";
import { Message, MessageAvatar, MessageContent } from "@/components/ui/message";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { cn } from "@/lib/utils";
import { isNote, type Message as ChatMessage } from "@/models/Chat";
import type { Person } from "@/models/Person";
import { parseQuestionMessage } from "@/models/Question";
import type { TrailBlock } from "@/utils/ThreadTrailUtility";

interface MessageRowProps {
  message: ChatMessage;
  author: Person;
  isOwn: boolean;
  // continuation says the previous message is the same person's, so this one drops its avatar and header.
  continuation?: boolean;
  // questionAnswered says the thread already moved past an Agent question, so its card is read-only.
  questionAnswered?: boolean;
  // trailBlock is the run that led to this message: its turns render above it, and its question is answered on the trail.
  trailBlock?: TrailBlock | undefined;
  // ticketId is the ticket a ticket thread belongs to, whose write permission edits and deletes its notes.
  ticketId?: string | undefined;
  onEdit: (messageId: string, body: string) => Promise<void>;
  onDelete: (messageId: string) => Promise<void>;
}

// timeOnHover is the clock time a grouped message of yours reveals beside its bubble, in place of the header it dropped.
interface MessageBubbleProps {
  message: ChatMessage;
  own: boolean;
  timeOnHover: boolean;
  actions: ReactNode;
}

const MessageBubble = ({ message, own, timeOnHover, actions }: MessageBubbleProps) => (
  <div
    data-slot="bubble"
    className={cn(
      "relative w-fit max-w-[75%] space-y-1.5 rounded-lg px-3 py-2 text-sm break-words",
      // Yours is the one bubble; everyone else's text runs plain under their name, so a long thread reads as a transcript.
      own && "bg-brand text-brand-foreground",
      !own && "max-w-[72ch] px-0 py-0",
    )}
  >
    {timeOnHover && (
      <MessageContinuationTime
        createdAt={message.created_at}
        className="absolute top-1/2 right-full mr-2 -translate-y-1/2 opacity-0 transition-opacity duration-150 ease-standard group-focus-within:opacity-100 group-hover:opacity-100"
      />
    )}
    {actions}
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
export const MessageRow = ({ message, author, isOwn, continuation = false, questionAnswered = false, trailBlock, ticketId, onEdit, onDelete }: MessageRowProps) => {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(message.body);
  const { open: confirmDelete } = useConfirmationDialog();

  if (message.deleted_at) return null;

  const isSystem = message.author_kind === "system";
  const isBot = message.author_kind === "bot";
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
    const confirmed = await confirmDelete({ message: "Delete this message?", confirmLabel: "Delete message" });
    if (confirmed) await onDelete(message.id);
  };

  return (
    <>
      {isSystem && <SystemMessageRow body={message.body} trailBlock={trailBlock} />}
      {isBot && <BotMessageRow message={message} continuation={continuation} />}
      {!isSystem && !isBot && (
        <Message align={align} className={cn("group px-3 py-0.5 transition-colors duration-150 ease-standard hover:bg-accent/40", message.pending && "opacity-60")}>
          {align === "start" && !continuation && (
            <MessageAvatar className="size-8 self-start bg-transparent">
              <MessageRowAvatar isAgent={isAgent} author={author} />
            </MessageAvatar>
          )}
          {align === "start" && continuation && (
            <MessageAvatar className="w-8 self-center overflow-visible bg-transparent">
              <MessageContinuationTime
                createdAt={message.created_at}
                className="opacity-0 transition-opacity duration-150 ease-standard group-focus-within:opacity-100 group-hover:opacity-100"
              />
            </MessageAvatar>
          )}
          <MessageContent>
            {!continuation && <MessageRowHeader align={align} message={message} isAgent={isAgent} author={author} />}
            {editing && (
              <MessageEditForm draft={draft} onDraftChange={setDraft} onCancel={() => setEditing(false)} onSave={() => void saveEdit()} />
            )}
            {!editing && !isAgent && (
              <MessageBubble
                message={message}
                own={align === "end"}
                timeOnHover={continuation && align === "end"}
                actions={
                  !message.pending && (
                    <MessageActions
                      message={message}
                      onEdit={canEditOrDelete ? startEdit : undefined}
                      onDelete={canEditOrDelete ? () => void remove() : undefined}
                    />
                  )
                }
              />
            )}
            {!editing && isAgent && <AgentMessageBody message={message} trailBlock={trailBlock} questionAnswered={questionAnswered} />}
            {!editing && isNote(message) && <NoteMessage message={message} ticketId={ticketId} />}
            {!editing && message.handoffs && message.handoffs.length > 0 && <HandoffPills handoffs={message.handoffs} />}
            {!editing && <MessageReactions message={message} />}
          </MessageContent>
        </Message>
      )}
    </>
  );
};
