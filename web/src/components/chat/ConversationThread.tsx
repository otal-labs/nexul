import { FileText, Hash, Ticket, Users, Volume2, type LucideIcon } from "lucide-react";
import { Suspense, useCallback, useRef } from "react";

import { ChatComposer } from "@/components/chat/ChatComposer";
import { ChatPaneState } from "@/components/chat/ChatPaneState";
import { LazyVoiceCallSection } from "@/components/chat/LazyVoiceCallSection";
import { MessageList } from "@/components/chat/MessageList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchMe } from "@/hooks/AuthHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import {
  useDeleteMessage,
  useEditMessage,
  useFetchMessages,
  useInterruptAgentTurn,
  useMarkChatRead,
  usePostMessage,
} from "@/hooks/ChatHooks";
import { useFetchDoc } from "@/hooks/DocHooks";
import { useFetchTicket } from "@/hooks/TicketHooks";
import { conversationLabel, type Conversation } from "@/models/Chat";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

interface ConversationThreadProps {
  workspaceId: string;
  conversation: Conversation;
  /** Default true; a ticket page hides it since the ticket's own header names the thread. */
  showHeader?: boolean;
}

const CONVERSATION_ICONS: Partial<Record<Conversation["kind"], LucideIcon>> = {
  voice_channel: Volume2,
  dm: Users,
  doc_thread: FileText,
  ticket_thread: Ticket,
};

export const ConversationThread = ({ workspaceId, conversation, showHeader = true }: ConversationThreadProps) => {
  const { data: me } = useFetchMe();
  const { data: messages, error, isPending } = useFetchMessages(conversation.id);
  const resolvePerson = usePersonLookup(workspaceId);
  const postMessage = usePostMessage(conversation.id);
  const editMessage = useEditMessage();
  const deleteMessage = useDeleteMessage(conversation.id);
  const interruptAgent = useInterruptAgentTurn(conversation.id);
  const markRead = useMarkChatRead(workspaceId);
  // A doc or ticket thread is titled with what it is about, like its sidebar row.
  const { data: doc } = useFetchDoc(conversation.kind === "doc_thread" ? conversation.doc_id : undefined);
  const { data: ticket } = useFetchTicket(conversation.kind === "ticket_thread" ? conversation.ticket_id : undefined);
  const label = doc?.title ?? ticket?.title ?? conversationLabel(conversation, { currentUserId: me?.user.id, resolvePerson });
  const isVoiceChannel = conversation.kind === "voice_channel";
  const isCallActive = useVoiceCallStore((s) => s.activeConversationId === conversation.id);

  // Fires when the newest message is actually seen, not on open, so a scrolled-up chat keeps its unread count.
  const lastMarkedRef = useRef<string | null>(null);
  const onNewestSeen = useCallback(
    (messageId: string) => {
      if (lastMarkedRef.current === messageId) return;
      lastMarkedRef.current = messageId;
      markRead.mutate(conversation.id);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [conversation.id],
  );

  const Icon = CONVERSATION_ICONS[conversation.kind] ?? Hash;

  return (
    <div className="flex h-full min-h-0 flex-col">
      {showHeader && (
        <div className="flex h-14 shrink-0 items-center gap-2 border-b border-border px-4">
          <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
          <span dir="auto" title={label} className="min-w-0 truncate text-sm font-semibold">
            {label}
          </span>
        </div>
      )}
      {isVoiceChannel && (
        <Suspense fallback={<LoadingDisplay label="Loading voice…" />}>
          <LazyVoiceCallSection conversation={conversation} active={isCallActive} />
        </Suspense>
      )}
      {isPending && (
        <ChatPaneState>
          <LoadingDisplay label="Loading messages…" />
        </ChatPaneState>
      )}
      {error && (
        <ChatPaneState>
          <ErrorDisplay error={error} title="Failed to load messages." />
        </ChatPaneState>
      )}
      {messages && (
        <MessageList
          // What counts as already seen restarts with each conversation.
          key={conversation.id}
          conversation={conversation}
          messages={messages}
          onNewestSeen={onNewestSeen}
          currentUserId={me?.user.id}
          resolveAuthor={resolvePerson}
          onEdit={async (messageId, body) => {
            await editMessage.mutateAsync({ messageId, body });
          }}
          onDelete={async (messageId) => {
            await deleteMessage.mutateAsync(messageId);
          }}
          onInterruptAgent={() => void interruptAgent.mutateAsync()}
        />
      )}
      <ChatComposer
        workspaceId={workspaceId}
        conversationId={conversation.id}
        placeholder={doc || ticket ? "Message the thread…" : `Message ${label}…`}
        onSend={async (body) => {
          await postMessage.mutateAsync(body);
        }}
      />
    </div>
  );
};
