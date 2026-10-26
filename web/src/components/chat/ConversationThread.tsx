import { FileText, Hash, Ticket as TicketIcon, Users, Volume2, type LucideIcon } from "lucide-react";
import { Suspense, useCallback, useRef } from "react";
import { Link } from "react-router";

import { conversationLabel, type Conversation } from "@nexul/client-core/chat";

import { CalendarTitleContext } from "@/components/chat/calendarTitle";
import { ChatComposer } from "@/components/chat/ChatComposer";
import { ChatPaneState } from "@/components/chat/ChatPaneState";
import { LazyVoiceCallSection } from "@/components/chat/LazyVoiceCallSection";
import { MessageList } from "@/components/chat/MessageList";
import { ReadOnlyComposer } from "@/components/chat/ReadOnlyComposer";
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
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchTicket } from "@/hooks/TicketHooks";
import { useChannelPostRefusal } from "@/hooks/useChannelPosting";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
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
  const wsPath = useWorkspacePath();
  const subjectPath = subjectPathOf(doc, ticket, projects);
  const isVoiceChannel = conversation.kind === "voice_channel";
  const isCallActive = useVoiceCallStore((s) => s.activeConversationId === conversation.id);
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

  const Icon = CONVERSATION_ICONS[conversation.kind] ?? Hash;

  return (
    <CalendarTitleContext value={label}>
      <div className="flex h-full min-h-0 flex-col">
        {showHeader && (
          <div className="h-14 shrink-0 border-b border-border">
            <div className="mx-auto flex h-full w-full max-w-3xl items-center gap-2 px-3">
              <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
              {subjectPath && (
                <Link
                  to={wsPath(subjectPath)}
                  dir="auto"
                  title={label}
                  className="min-w-0 truncate text-sm font-semibold underline-offset-4 hover:underline"
                >
                  {label}
                </Link>
              )}
              {!subjectPath && (
                <span dir="auto" title={label} className="min-w-0 truncate text-sm font-semibold">
                  {label}
                </span>
              )}
            </div>
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
          onSend={async (body) => {
            await postMessage.mutateAsync(body);
          }}
        />}
      </div>
    </CalendarTitleContext>
  );
};
