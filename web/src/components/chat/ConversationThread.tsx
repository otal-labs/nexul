import { ArrowLeft, Hash, Users, Volume2 } from "lucide-react";
import { Suspense, useCallback, useRef } from "react";

import { ChatComposer } from "@/components/chat/ChatComposer";
import { LazyVoiceCallSection } from "@/components/chat/LazyVoiceCallSection";
import { MessageList } from "@/components/chat/MessageList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { useFetchMe } from "@/hooks/AuthHooks";
import {
  useChatAuthorLookup,
  useDeleteMessage,
  useEditMessage,
  useFetchMessages,
  useInterruptAgentTurn,
  useMarkChatRead,
  usePostMessage,
} from "@/hooks/ChatHooks";
import { conversationLabel, type Conversation } from "@/models/Chat";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

interface ConversationThreadProps {
  workspaceId: string;
  conversation: Conversation;
  /** Renders a back button in this component's own header on narrow widths. */
  onBack?: () => void;
  /** Default true; a ticket page hides it since the ticket's own header names the thread. */
  showHeader?: boolean;
}

const conversationIcon = (kind: Conversation["kind"]) => {
  if (kind === "voice_channel") return Volume2;
  if (kind === "dm") return Users;
  return Hash;
};

export const ConversationThread = ({ workspaceId, conversation, onBack, showHeader = true }: ConversationThreadProps) => {
  const { data: me } = useFetchMe();
  const { data: messages, error, isPending } = useFetchMessages(conversation.id);
  const resolveAuthorLogin = useChatAuthorLookup(workspaceId);
  const postMessage = usePostMessage(conversation.id);
  const editMessage = useEditMessage();
  const deleteMessage = useDeleteMessage(conversation.id);
  const interruptAgent = useInterruptAgentTurn(conversation.id);
  const markRead = useMarkChatRead(workspaceId);
  const label = conversationLabel(conversation, { currentUserId: me?.user.id, resolveLogin: resolveAuthorLogin });
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

  const Icon = conversationIcon(conversation.kind);

  return (
    <div className="flex h-full min-h-0 flex-col">
      {showHeader && (
        <div className="flex h-14 shrink-0 items-center gap-2 border-b border-border px-4">
          {onBack && (
            <Button size="icon" variant="ghost" className="size-7 -ml-1" aria-label="Back to conversations" onClick={onBack}>
              <ArrowLeft className="size-4" aria-hidden />
            </Button>
          )}
          <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
          <span className="min-w-0 truncate text-sm font-semibold">{label}</span>
        </div>
      )}
      {isVoiceChannel && (
        <Suspense fallback={<LoadingDisplay label="Loading voice…" />}>
          <LazyVoiceCallSection workspaceId={workspaceId} conversation={conversation} active={isCallActive} />
        </Suspense>
      )}
      {isPending && <LoadingDisplay label="Loading messages…" />}
      {error && <ErrorDisplay error={error} title="Failed to load messages." />}
      {messages && (
        <MessageList
          conversation={conversation}
          messages={messages}
          onNewestSeen={onNewestSeen}
          currentUserId={me?.user.id}
          resolveAuthorLogin={resolveAuthorLogin}
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
        placeholder={`Message ${label}…`}
        onSend={async (body) => {
          await postMessage.mutateAsync(body);
        }}
      />
    </div>
  );
};
