import type { QueryClient } from "@tanstack/react-query";

import {
  followConversationDeleted,
  getChatConversationsKey,
  getChatThreadIndicatorsKey,
  getChatTicketThreadStatusKey,
  getChatUnreadKey,
  markCachedMessageDeleted,
  upsertCachedMessage,
} from "@/hooks/ChatHooks";
import { useAgentStreamStore } from "@/stores/agentStreamStore";
import { isNote, type Conversation, type ConversationDeleted, type Message } from "@/models/Chat";
import type { Handoff } from "@/models/Handoff";
import { parseQuestionMessage } from "@/models/Question";
import type { ActivityKind, RunFrame } from "@/models/Trail";
import type { LiveFollower } from "@/lib/live";

// A full-text-replace snapshot of the in-progress @Agent turn bubble, keyed by conversation + message id.
interface AgentStreamPayload {
  conversation_id: string;
  message_id: string;
  text: string;
  streaming: boolean;
  activity?: string;
  activity_kind?: ActivityKind;
  activity_tool?: string;
  // Set only on the frame for the one hand-off that changed.
  handoff?: Handoff;
}

// Message frames carry the whole message, so they patch the cached list instead of triggering a refetch.
interface MessagePayload {
  message: Message;
}

interface MessageDeletedPayload {
  conversation_id: string;
  message_id: string;
  deleted_at: string;
}

// Conversation frames after the created one name the conversation and its workspace (internal/chat/events.go).
interface ConversationPlace {
  conversation_id: string;
  workspace_id: string;
}

const refetch = (client: QueryClient, queryKey: unknown[]) => client.invalidateQueries({ queryKey, exact: true });

// A message frame names no workspace; the conversation lists the viewer has loaded say which one holds it.
const holdingWorkspace = (client: QueryClient, conversationId: string) =>
  client
    .getQueriesData<Conversation[]>({ queryKey: [getChatConversationsKey] })
    .find(([, list]) => list?.some((c) => c.id === conversationId))?.[0][1] as string | undefined;

// Unread counts follow every message, in the workspace whose list holds its conversation, or every one when none does.
const refetchUnread = (client: QueryClient, workspaceId: string | undefined) =>
  client.invalidateQueries({ queryKey: workspaceId ? [getChatUnreadKey, workspaceId] : [getChatUnreadKey], exact: !!workspaceId });

const refetchWorkspace = (client: QueryClient, workspaceId: string) =>
  Promise.all([refetch(client, [getChatConversationsKey, workspaceId]), refetch(client, [getChatUnreadKey, workspaceId])]);

// A question mid-turn takes the bubble's text, not the hand-offs still working, so their pills stay until the reply.
const yieldStream = (message: Message) => {
  const store = useAgentStreamStore.getState();
  const stream = store.streams[message.conversation_id];
  if (stream && stream.handoffs.length > 0 && parseQuestionMessage(message.body)) {
    store.setStream(message.conversation_id, { messageId: stream.messageId, text: "", streaming: stream.streaming });
    return;
  }
  store.clearStream(message.conversation_id);
};

const markThread = (client: QueryClient, queryKey: unknown[], ticketId: string) =>
  client.setQueriesData<Record<string, boolean>>({ queryKey }, (threads) => threads && { ...threads, [ticketId]: true });

// A ticket's thread now exists: its card on the board and its Thread section show it without a request.
const ticketThreadStarted = (client: QueryClient, ticketId: string, projectId?: string) => {
  markThread(client, projectId ? [getChatThreadIndicatorsKey, projectId] : [getChatThreadIndicatorsKey], ticketId);
  markThread(client, [getChatTicketThreadStatusKey, ticketId], ticketId);
};

const followMessage = ({ message }: MessagePayload, { client }: { client: QueryClient }) => {
  upsertCachedMessage(client, message);
  // Once the turn's real message lands (author_kind "agent"), the ephemeral stream bubble yields to it; a note does not end it.
  if (message.author_kind === "agent" && !isNote(message)) yieldStream(message);
};

export const chatFollower: LiveFollower = {
  "chat.conversation.created": ({ conversation }: { conversation: Conversation }, { client }) => {
    if (conversation.ticket_id) ticketThreadStarted(client, conversation.ticket_id, conversation.project_id);
    return refetch(client, [getChatConversationsKey, conversation.workspace_id]);
  },
  "chat.conversation.updated": ({ workspace_id }: ConversationPlace, { client }) => refetch(client, [getChatConversationsKey, workspace_id]),
  "chat.conversation.members_changed": ({ workspace_id }: ConversationPlace, { client }) => refetchWorkspace(client, workspace_id),
  "chat.conversation.deleted": (deleted: ConversationDeleted, { client, navigate, location }) => {
    followConversationDeleted(navigate, location, deleted);
    client.setQueryData<Conversation[]>([getChatConversationsKey, deleted.workspace_id], (list) => list?.filter((c) => c.id !== deleted.conversation_id));
    return refetch(client, [getChatUnreadKey, deleted.workspace_id]);
  },
  // A message in a conversation no loaded list holds is the viewer's first sight of it, so the lists refetch too.
  "chat.message.created": (payload: MessagePayload, live) => {
    followMessage(payload, live);
    const workspaceId = holdingWorkspace(live.client, payload.message.conversation_id);
    return Promise.all([refetchUnread(live.client, workspaceId), !workspaceId && live.client.invalidateQueries({ queryKey: [getChatConversationsKey] })]);
  },
  "chat.message.updated": followMessage,
  "chat.message.deleted": ({ conversation_id: conversationId, message_id: messageId, deleted_at: deletedAt }: MessageDeletedPayload, { client }) => {
    markCachedMessageDeleted(client, conversationId, messageId, deletedAt);
    return refetchUnread(client, holdingWorkspace(client, conversationId));
  },
  "chat.agent.stream": (p: AgentStreamPayload) => {
    // An empty non-streaming frame is the pipeline's clear signal, or the bubble spins forever.
    if (!p.streaming && !p.text) {
      useAgentStreamStore.getState().clearStream(p.conversation_id);
      return;
    }
    useAgentStreamStore.getState().setStream(p.conversation_id, {
      messageId: p.message_id,
      text: p.text,
      streaming: p.streaming,
      activity: p.activity ?? "",
      ...(p.activity_kind && { activityKind: p.activity_kind }),
      ...(p.activity_tool && { activityTool: p.activity_tool }),
      ...(p.handoff && { handoff: p.handoff }),
    });
  },
  // A ticket run that is starting has opened its thread, so the Thread section shows Started in place of Start thread.
  "play.run": (run: RunFrame, { client }) => {
    if (run.target_type !== "ticket" || run.state !== "starting") return;
    markThread(client, [getChatTicketThreadStatusKey, run.target_id], run.target_id);
  },
};
