import { FileText, Hash, Lock, Ticket as TicketIcon, Users, Volume2, type LucideIcon } from "lucide-react";
import { Suspense, useCallback, useRef } from "react";

import { conversationLabel, type Conversation } from "@nexul/client-core/chat";

import { CalendarTitleContext } from "@/components/chat/calendarTitle";
import { ChatComposer } from "@/components/chat/ChatComposer";
import { ConversationHeader } from "@/components/chat/ConversationHeader";
import { ChatPaneState } from "@/components/chat/ChatPaneState";
import { LazyVoiceCallSection } from "@/components/chat/LazyVoiceCallSection";
import { MessageList } from "@/components/chat/MessageList";
import { ReadOnlyComposer } from "@/components/chat/ReadOnlyComposer";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { VoiceLobbySlot } from "@/components/voice/VoiceLobbyRow";
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
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchTicket } from "@/hooks/TicketHooks";
import { useChannelPostRefusal } from "@/hooks/useChannelPosting";
import { cn } from "@/lib/utils";
import { useVoiceCallStore } from "@/stores/voiceCallStore";
import { channelMention } from "@/models/Chat";
import { type Doc } from "@/models/Doc";
import { docPath, projectTokenById, type Project } from "@/models/Project";
import { ticketPath, type Ticket } from "@/models/Ticket";

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
  ticket_thread: TicketIcon,
};

// A doc or ticket thread's title leads back to what it is about.
const subjectPathOf = (
  doc: Doc | undefined,
  ticket: Ticket | undefined,
  projects: Project[] | undefined,
): string | undefined => {
  if (!projects) return undefined;
  if (doc) return docPath(projectTokenById(projects, doc.project_id), doc.id);
  if (ticket) return ticketPath(ticket, projects.find((p) => p.id === ticket.project_id)?.prefix);
  return undefined;
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
  const { data: projects } = useFetchProjects();
  const subjectPath = subjectPathOf(doc, ticket, projects);
  const isVoiceChannel = conversation.kind === "voice_channel";
  const isCallActive = useVoiceCallStore((s) => s.activeConversationId === conversation.id);
  const stageBeside = useVoiceCallStore((s) => isCallActive && s.status === "connected" && !!s.room && !s.stageCollapsed);
  const postRefusal = useChannelPostRefusal(conversation);

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

  // Every message row is memoized on these, so they keep one identity per conversation.
  const { mutateAsync: edit } = editMessage;
  const { mutateAsync: remove } = deleteMessage;
  const onEdit = useCallback(async (messageId: string, body: string) => {
    await edit({ messageId, body });
  }, [edit]);
  const onDelete = useCallback(async (messageId: string) => {
    await remove(messageId);
  }, [remove]);

  // A private channel leads with its lock, as its sidebar row does.
  const Icon = conversation.private ? Lock : (CONVERSATION_ICONS[conversation.kind] ?? Hash);

  return (
    <CalendarTitleContext value={label}>
      <div className="@container/thread flex h-full min-h-0 flex-col">
        {showHeader && <ConversationHeader conversation={conversation} label={label} icon={Icon} subjectPath={subjectPath} inCall={stageBeside} />}
        {/* With the stage showing, it and the chat sit side by side from 1000px, the chat a 380px column. */}
        <div className={cn("flex min-h-0 flex-1 flex-col", stageBeside && "@[1000px]/thread:flex-row")}>
        {isVoiceChannel && (
          <Suspense fallback={<VoiceLobbySlot />}>
            <LazyVoiceCallSection conversation={conversation} active={isCallActive} />
          </Suspense>
        )}
        <div className={cn("flex min-h-0 flex-1 flex-col", stageBeside && "@[1000px]/thread:w-95 @[1000px]/thread:flex-none @[1000px]/thread:border-l @[1000px]/thread:border-border")}>
        {isPending && (
          <ChatPaneState>
            <LoadingDisplay label="Loading messages…" />
          </ChatPaneState>
        )}
        {error && (
          <ChatPaneState>
            <ErrorDisplay error={error} title="Couldn't load messages." />
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
            onEdit={onEdit}
            onDelete={onDelete}
            onInterruptAgent={() => void interruptAgent.mutateAsync()}
          />
        )}
        {postRefusal && <ReadOnlyComposer label={channelMention(conversation)} source={postRefusal} />}
        {!postRefusal && <ChatComposer
          workspaceId={workspaceId}
          conversationId={conversation.id}
          placeholder={doc || ticket ? "Message the thread…" : `Message ${label}…`}
          onSend={async (body, queue) => {
            await postMessage.mutateAsync({ body, queue });
          }}
        />}
        </div>
        </div>
      </div>
    </CalendarTitleContext>
  );
};
