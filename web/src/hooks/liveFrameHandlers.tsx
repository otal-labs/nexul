import type { QueryClient } from "@tanstack/react-query";

import type { ServerFrame } from "@/api/ws";
import { CanvasFrameSchema } from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";
import { docChanged, getDocWatchersKey } from "@/hooks/DocHooks";
import { getInterviewAnswersKey, getMemoriesKey } from "@/hooks/MemoryHooks";
import { getInterviewSourcesKey } from "@/hooks/InterviewSourceHooks";
import { getMeKey } from "@/hooks/AuthHooks";
import { getAttachmentsKey } from "@/hooks/AttachmentHooks";
import { getNoteTextKey } from "@/hooks/NoteHooks";
import { ticketChanged, ticketCreated, ticketRemoved } from "@/hooks/TicketCache";
import { getTicketLinksKey } from "@/hooks/TicketHooks";
import {
  getChatThreadIndicatorsKey,
  getChatTicketThreadStatusKey,
  markCachedMessageDeleted,
  upsertCachedMessage,
} from "@/hooks/ChatHooks";
import { applyCachedReaction } from "@/hooks/ReactionHooks";
import { setCachedTunnelStatus, type TunnelStatusChangedPayload } from "@/hooks/PairingHooks";
import { getActiveTrailsKey, getTrailKey, getTrailsKey } from "@/hooks/TrailHooks";
import { useAgentStreamStore } from "@/stores/agentStreamStore";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useDeviceArrivalStore } from "@/stores/deviceArrivalStore";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import { useVoiceOccupancyStore } from "@/stores/voiceOccupancyStore";
import { isNote, type Conversation, type ConversationDeleted, type Message } from "@/models/Chat";
import type { Doc, DocWatchers } from "@/models/Doc";
import type { Handoff } from "@/models/Handoff";
import { parseQuestionMessage } from "@/models/Question";
import type { Ticket } from "@/models/Ticket";
import { isTrailActive, type ActivityKind, type RunFrame } from "@/models/Trail";
import type { MeResponse, SessionClient } from "@/models/User";
import type { VoiceOccupant } from "@/models/Voice";
import type { WorkspaceUpdate } from "@/models/Workspace";

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

interface MessageReactionsChangedPayload {
  conversation_id: string;
  message_id: string;
  user_id: string;
  emoji: string;
  reacted: boolean;
}

interface MessageDeletedPayload {
  conversation_id: string;
  message_id: string;
  deleted_at: string;
}

// The running setup turn's latest step as one line, for the commentary under its row.
interface SetupTurnActivityPayload {
  turn_id: string;
  status: string;
  call_id?: string;
  kind?: ActivityKind;
  tool?: string;
  text?: string;
  at?: string;
}

// The metadata of a session that just signed in; never the token.
interface SessionCreatedPayload {
  session_id: string;
  user_id: string;
  client: SessionClient;
  platform: string;
  label: string;
}

// One voice channel's full occupant list after a change, applied wholesale.
interface OccupancyChangedPayload {
  conversation_id: string;
  occupants: VoiceOccupant[];
}

// Ticket frames carry the whole ticket after the change (internal/tickets/events.go).
interface TicketPayload {
  ticket: Ticket;
}

interface DocUpdatedPayload {
  doc: Doc;
  actor_id?: string;
}

export interface RouterFollowers {
  onWorkspaceUpdated: (update: WorkspaceUpdate) => void;
  onConversationDeleted: (deleted: ConversationDeleted) => void;
}

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

type FrameHandler = (frame: ServerFrame, client: QueryClient, router: RouterFollowers) => void;

const onTicketChanged: FrameHandler = (frame, client) => void ticketChanged(client, (frame.payload as TicketPayload).ticket);

const onMessage: FrameHandler = (frame, client) => {
  const p = frame.payload as MessagePayload;
  if (!p.message) return;
  upsertCachedMessage(client, p.message);
  // Once the turn's real message lands (author_kind "agent"), the ephemeral stream bubble yields to it; a note does not end it.
  if (p.message.author_kind === "agent" && !isNote(p.message)) yieldStream(p.message);
  // A note's file changed under its message: open renders and the pill's size follow it; an open editor follows its room.
  if (frame.topic === "chat.message.updated" && isNote(p.message)) {
    void client.invalidateQueries({ queryKey: [getNoteTextKey, p.message.attachment_id] });
    void client.invalidateQueries({ queryKey: [getAttachmentsKey, { conversation_id: p.message.conversation_id }] });
  }
};

// Frames the browser applies itself; a topic may also list queries to refetch in pushTopics.
export const frameHandlers: Record<string, FrameHandler> = {
  "workspace.updated": (frame, _, router) => router.onWorkspaceUpdated(frame.payload as WorkspaceUpdate),
  "chat.conversation.deleted": (frame, _, router) => router.onConversationDeleted(frame.payload as ConversationDeleted),
  // A new ticket thread shows on the board's card and on its ticket's Thread section, in every open browser.
  "chat.conversation.created": (frame, client) => {
    const ticketId = (frame.payload as { conversation?: Conversation }).conversation?.ticket_id;
    if (!ticketId) return;
    void client.invalidateQueries({ queryKey: [getChatThreadIndicatorsKey] });
    void client.invalidateQueries({ queryKey: [getChatTicketThreadStatusKey, ticketId] });
  },
  topology: (frame) => {
    const flow = useFlowStore.getState();
    const parsed = CanvasFrameSchema.safeParse(frame.payload);
    if (parsed.success && parsed.data.workspace_id === flow.workspaceId) flow.applyServerPatch(parsed.data);
  },
  "chat.agent.stream": (frame) => {
    const p = frame.payload as AgentStreamPayload;
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
  "play.run": (frame, client) => {
    const p = frame.payload as RunFrame;
    const previous = usePlayRunStore.getState().frames[p.trail_id]?.state;
    usePlayRunStore.getState().applyFrame(p);
    // Activity lines only move the store; a state change refetches the rows and the board's active set.
    if (previous === p.state) return;
    void client.invalidateQueries({ queryKey: [getTrailsKey, p.target_type, p.target_id] });
    void client.invalidateQueries({ queryKey: [getTrailKey, p.trail_id] });
    void client.invalidateQueries({ queryKey: [getActiveTrailsKey] });
    // A finished interview run has written the memory and recorded its rounds; the page shows both without a reload.
    if (p.target_type === "interview" && !isTrailActive(p.state)) {
      void client.invalidateQueries({ queryKey: [getMemoriesKey] });
      void client.invalidateQueries({ queryKey: [getInterviewAnswersKey, p.target_id] });
    }
    if (p.target_type !== "ticket") return;
    // Starting shows the Thread section's Started message in place of "Start thread" without a reload.
    if (p.state === "starting") void client.invalidateQueries({ queryKey: [getChatTicketThreadStatusKey, p.target_id] });
    // A done run's new column and linked branch/PR show up without a reload.
    if (!isTrailActive(p.state)) {
      void ticketChanged(client, p.target_id);
      void client.invalidateQueries({ queryKey: [getTicketLinksKey, p.target_id] });
    }
  },
  "chat.message.created": onMessage,
  "chat.message.updated": onMessage,
  "chat.message.deleted": (frame, client) => {
    const p = frame.payload as MessageDeletedPayload;
    if (p.message_id) markCachedMessageDeleted(client, p.conversation_id, p.message_id, p.deleted_at);
  },
  "chat.message.reactions_changed": (frame, client) => {
    const p = frame.payload as MessageReactionsChangedPayload;
    applyCachedReaction(client, p.conversation_id, { messageId: p.message_id, userId: p.user_id, emoji: p.emoji, reacted: p.reacted });
  },
  "computer.tunnel_status_changed": (frame, client) => setCachedTunnelStatus(client, frame.payload as TunnelStatusChangedPayload),
  "computer.setup_turn_activity": (frame) => {
    const p = frame.payload as SetupTurnActivityPayload;
    useSetupActivityStore
      .getState()
      .push(p.turn_id, { kind: p.kind ?? "other", call_id: p.call_id ?? "", tool: p.tool ?? "", summary: p.status, detail: p.text ?? "", at: p.at ?? "" });
  },
  "session.created": (frame, client) => {
    const p = frame.payload as SessionCreatedPayload;
    // Every browser hears every session; only the viewer's own phone flips their Devices page.
    const me = client.getQueryData<MeResponse>([getMeKey]);
    if (p.client === "phone" && p.user_id === me?.user.id) {
      useDeviceArrivalStore.getState().arrive({ id: p.session_id, platform: p.platform, label: p.label });
    }
  },
  "voice.occupancy.changed": (frame) => {
    const p = frame.payload as OccupancyChangedPayload;
    useVoiceOccupancyStore.getState().setChannel(p.conversation_id, p.occupants);
  },
  "ticket.created": (frame, client) => void ticketCreated(client, (frame.payload as TicketPayload).ticket),
  "ticket.updated": onTicketChanged,
  "ticket.status_changed": onTicketChanged,
  "ticket.developer_changed": onTicketChanged,
  "ticket.tester_changed": onTicketChanged,
  "ticket.finished": onTicketChanged,
  // A category move re-appends the ticket server-side, and the frame names only its id.
  "ticket.category_changed": (frame, client) => void ticketChanged(client, (frame.payload as { ticket_id: string }).ticket_id),
  "ticket.deleted": (frame, client) => void ticketRemoved(client, (frame.payload as { id: string }).id),
  "doc.updated": (frame, client) => {
    const { doc, actor_id: actorID } = frame.payload as DocUpdatedPayload;
    docChanged(client, doc);
    // An edit makes its editor a watcher; one already watching changes nothing there.
    const watchers = client.getQueryData<DocWatchers>([getDocWatchersKey, doc.id]);
    if (actorID && !watchers?.watchers.some((w) => w.user_id === actorID)) {
      void client.invalidateQueries({ queryKey: [getDocWatchersKey, doc.id], exact: true });
    }
    // An interview source names the doc and dates its change.
    void client.invalidateQueries({ queryKey: [getInterviewSourcesKey, doc.project_id], exact: true });
  },
};
