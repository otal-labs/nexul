import { useEffect, useRef } from "react";
import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router";

import { CanvasFrameSchema } from "@/models/Topology";
import {
  LiveEventsClient,
  type LiveEventsClientOptions,
  type ServerFrame,
} from "@/api/ws";
import { useFlowStore } from "@/stores/flowStore";
import { getDeployKey, getDeployLogKey } from "@/hooks/DeployHooks";
import { getDnsExposuresKey, getDnsGatewaysKey } from "@/hooks/DnsHooks";
import { getDocClarificationKey, getDocKey, getDocsKey, getDocWatchersKey } from "@/hooks/DocHooks";
import { getDocFoldersKey } from "@/hooks/DocFolderHooks";
import { getInstanceUpgradeKey } from "@/hooks/InstanceUpgradeHooks";
import { getProjectPeopleKey, getWorkspacePeopleKey } from "@/hooks/PeopleHooks";
import { getProjectAccessKey } from "@/hooks/ProjectHooks";
import { followWorkspaceUpdate, getMyRoleKey, getWorkspacesKey } from "@/hooks/WorkspaceHooks";
import { getTeamKey } from "@/models/Team";
import type { MyWorkspaceInfo } from "@/models/Permission";
import { getInterviewAnswersKey, getMemoriesKey, getMemoryKey, getMemoryVersionsKey } from "@/hooks/MemoryHooks";
import { getInterviewDraftsKey, getInterviewSourcesKey } from "@/hooks/InterviewSourceHooks";
import { getNotificationsKey, getUnreadCountKey } from "@/hooks/NotificationHooks";
import { getMeKey, getPATsKey, getSessionsKey } from "@/hooks/AuthHooks";
import { getAttachmentsKey } from "@/hooks/AttachmentHooks";
import { getNoteTextKey } from "@/hooks/NoteHooks";
import { getComputerSetupKey } from "@/hooks/ComputerSetupHooks";
import {
  getComputersKey,
  getHarnessProvidersKey,
  getHarnessResolveKey,
  getMCPTokenKey,
} from "@/hooks/PairingHooks";
import { getWorkspaceRolesKey } from "@/hooks/RoleHooks";
import { getRunnerQueueKey, getRunnersKey } from "@/hooks/RunnerHooks";
import { getServiceDeploysKey, getServicesKey } from "@/hooks/ServiceHooks";
import { getStackDeploysKey } from "@/hooks/StackHooks";
import { getCategoriesKey, getProjectCategoriesKey } from "@/hooks/CategoryHooks";
import { getProjectTicketTypesKey, getTicketTypesKey } from "@/hooks/TicketTypeHooks";
import { getProjectStatusesKey, getStatusesKey } from "@/hooks/StatusHooks";
import { getTicketKey, getTicketLinksKey, getTicketsKey } from "@/hooks/TicketHooks";
import { getBlockersKey, getTicketLinkSetKey } from "@/hooks/TicketLinkHooks";
import {
  followConversationDeleted,
  getChatConversationsKey,
  getChatTicketThreadStatusKey,
  getChatUnreadKey,
  markCachedMessageDeleted,
  upsertCachedMessage,
} from "@/hooks/ChatHooks";
import { applyCachedReaction } from "@/hooks/ReactionHooks";
import { isNote, type ConversationDeleted, type Message } from "@/models/Chat";
import type { Handoff } from "@/models/Handoff";
import { parseQuestionMessage } from "@/models/Question";
import type { MeResponse, SessionClient } from "@/models/User";
import { getServerVersionKey, notifyIfServerUpdated } from "@/hooks/VersionHooks";
import { setCachedTunnelStatus, type TunnelStatusChangedPayload } from "@/hooks/PairingHooks";
import { getApplicablePlaysKey, getWorkspacePlaysKey } from "@/hooks/PlayHooks";
import { TEMPLATE_QUERY_KEYS } from "@/hooks/TemplateHooks";
import { getActiveTrailsKey, getTrailKey, getTrailsKey } from "@/hooks/TrailHooks";
import { useAgentStreamStore } from "@/stores/agentStreamStore";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useDeviceArrivalStore } from "@/stores/deviceArrivalStore";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import { useVoiceOccupancyStore } from "@/stores/voiceOccupancyStore";
import { isTrailActive, type ActivityKind, type RunFrame } from "@/models/Trail";
import type { VoiceOccupant } from "@/models/Voice";
import type { WorkspaceUpdate } from "@/models/Workspace";

// Maps push topics to the queries they invalidate.
const pushTopics: Record<string, string[]> = {
  "runner.connected": [getRunnersKey],
  "runner.disconnected": [getRunnersKey],
  "deploy.build_started": [getRunnersKey, getRunnerQueueKey],
  "deploy.build_progress": [getRunnersKey],
  "deploy.build_completed": [getRunnersKey, getRunnerQueueKey],
  "deploy.deploy_progress": [getRunnersKey],
  "deploy.status_changed": [getRunnersKey, getRunnerQueueKey],
  // Deploy reads refetch only on deploy.updated: the runner topics above fire before the deploy domain has
  // committed, so a refetch on them can read the record from before the change.
  // ponytail: whole-log refetch per batch (≤ 4/s); append lines into the cache if logs get large.
  "deploy.updated": [getDeployKey, getDeployLogKey, getStackDeploysKey, getServiceDeploysKey],
  "service.created": [getServicesKey],
  "service.updated": [getServicesKey],
  "service.deleted": [getServicesKey],
  "dns.gateway_changed": [getDnsGatewaysKey],
  "dns.exposure_changed": [getDnsExposuresKey],
  "notification.created": [getNotificationsKey, getUnreadCountKey],
  // The pickers' "needs setup" tags follow a setup turn confirming or withdrawing a provider.
  "computer.setup_confirmed": [getHarnessProvidersKey, getComputerSetupKey],
  "computer.setup_unconfirmed": [getHarnessProvidersKey, getComputerSetupKey],
  // The Set up step's rows and each computer row's provider lines follow a run turn by turn.
  "computer.setup_turn_changed": [getComputerSetupKey],
  "computer.setup_finished": [getComputerSetupKey, getHarnessProvidersKey],
  // A computer row goes from pairing in progress to paired, or appears and leaves, without a refresh.
  "computer.paired": [getComputersKey, getHarnessResolveKey],
  // A computer whose T3 Code moved to its new orchestrator shows its new version without a refresh.
  "computer.harness_switched": [getComputersKey, getHarnessResolveKey],
  "computer.tunnel_created": [getComputersKey],
  "computer.tunnel_removed": [getComputersKey, getHarnessResolveKey],
  // A computer row's MCP token line follows a mint or revoke from setup, the row, or an MCP tool.
  "personal_access_token.minted": [getMCPTokenKey, getPATsKey],
  "personal_access_token.revoked": [getMCPTokenKey, getPATsKey],
  // The Devices list follows a phone connecting or a device being signed out, without a refresh.
  "session.created": [getSessionsKey],
  "session.revoked": [getSessionsKey],
  "category.created": [getCategoriesKey, getProjectCategoriesKey],
  "category.updated": [getCategoriesKey, getProjectCategoriesKey],
  "category.deleted": [getCategoriesKey, getProjectCategoriesKey],
  "doc.created": [getDocsKey],
  // An edit makes its editor a watcher, which the Watch control's count shows; an interview source names the doc and dates its change.
  "doc.updated": [getDocsKey, getDocKey, getDocWatchersKey, getInterviewSourcesKey],
  "doc.watchers.changed": [getDocWatchersKey],
  "doc.clarification.round_started": [getDocClarificationKey],
  "doc.clarification.round_posted": [getDocClarificationKey],
  "doc.clarification.round_ended": [getDocClarificationKey],
  "doc.clarification.round_answered": [getDocClarificationKey],
  "doc.clarification.answer_saved": [getDocClarificationKey],
  "doc.clarification.answer_cleared": [getDocClarificationKey],
  "doc.clarification.anything_else_saved": [getDocClarificationKey],
  "doc.clarification.closed": [getDocClarificationKey],
  // The inbox groups doc rows by the folder each doc is in now, so a move, rename, or delete regroups it.
  "doc.moved": [getDocsKey, getDocKey, getDocFoldersKey, getNotificationsKey],
  "doc.folder.created": [getDocFoldersKey],
  "doc.folder.updated": [getDocFoldersKey, getNotificationsKey],
  "doc.folder.deleted": [getDocFoldersKey, getDocsKey, getNotificationsKey],
  "ticket.created": [getTicketsKey],
  "ticket.updated": [getTicketsKey, getTicketKey, getTicketLinksKey, getTrailsKey, getTicketLinkSetKey, getBlockersKey],
  // A blocker reaching a done-stage column clears the blocked card the moment it moves.
  "ticket.status_changed": [getTicketsKey, getTicketKey, getTicketLinksKey, getTrailsKey, getTicketLinkSetKey, getBlockersKey],
  "ticket.link_created": [getTicketLinkSetKey, getBlockersKey],
  "ticket.link_deleted": [getTicketLinkSetKey, getBlockersKey],
  "ticket.developer_changed": [getTicketsKey, getTicketKey],
  "ticket.tester_changed": [getTicketsKey, getTicketKey],
  "ticket.finished": [getTicketsKey],
  "ticket.category_changed": [getTicketsKey, getTicketKey],
  "ticket_type.created": [getTicketTypesKey, getProjectTicketTypesKey],
  "ticket_type.updated": [getTicketTypesKey, getProjectTicketTypesKey],
  "ticket_type.deleted": [getTicketTypesKey, getProjectTicketTypesKey],
  "status.created": [getStatusesKey, getProjectStatusesKey, getTicketsKey],
  "status.updated": [getStatusesKey, getProjectStatusesKey, getTicketsKey, getTicketLinkSetKey, getBlockersKey],
  "status.deleted": [getStatusesKey, getProjectStatusesKey, getTicketsKey],
  "chat.conversation.created": [getChatConversationsKey],
  "chat.conversation.updated": [getChatConversationsKey],
  "chat.conversation.deleted": [getChatConversationsKey, getChatUnreadKey],
  // A channel switched private or public, or someone added or removed, appears in or drops from each reader's list.
  "chat.conversation.members_changed": [getChatConversationsKey, getChatUnreadKey],
  "chat.message.created": [getChatConversationsKey, getChatUnreadKey],
  "chat.message.deleted": [getChatUnreadKey],
  "instance.upgrade_changed": [getInstanceUpgradeKey, getServerVersionKey],
  "play.created": [getWorkspacePlaysKey, getApplicablePlaysKey],
  "play.updated": [getWorkspacePlaysKey, getApplicablePlaysKey],
  "play.deleted": [getWorkspacePlaysKey, getApplicablePlaysKey],
  // An instance template changes what every unedited workspace shows and what each copy is compared with.
  "instance_template.updated": TEMPLATE_QUERY_KEYS,
  "memory.created": [getMemoriesKey],
  "memory.updated": [getMemoriesKey, getMemoryKey, getMemoryVersionsKey, getInterviewSourcesKey],
  "memory.deleted": [getMemoriesKey, getMemoryKey, getInterviewSourcesKey],
  "interview_answer.saved": [getInterviewAnswersKey],
  "interview_answer.cleared": [getInterviewAnswersKey],
  "interview_source.added": [getInterviewSourcesKey],
  "interview_source.changed": [getInterviewSourcesKey],
  "interview_source.removed": [getInterviewSourcesKey],
  "interview_draft.saved": [getInterviewDraftsKey],
  "interview_draft.dismissed": [getInterviewDraftsKey],
  // The Team list and a person's detail follow account and membership changes made anywhere, MCP included.
  "account.admitted": [getTeamKey],
  "account.disabled": [getTeamKey],
  "account.reactivated": [getTeamKey],
  "account.removed": [getTeamKey, getWorkspacePeopleKey],
  "account.restored": [getTeamKey],
  // Someone's first socket opening or last one closing; the frame names nobody, the refetch applies the Team's scoping.
  "account.presence_changed": [getTeamKey],
  // A new name or picture reaches every open screen that shows the person, the saver's other devices included.
  "account.profile_updated": [getWorkspacePeopleKey, getTeamKey, getMeKey],
  "workspace.member.added": [getTeamKey, getWorkspacePeopleKey, getWorkspacesKey],
  "workspace.member.removed": [getTeamKey, getWorkspacePeopleKey, getWorkspacesKey],
  "workspace.member.updated": [getTeamKey, getProjectAccessKey, getProjectPeopleKey],
  // Someone else's Project access moved: who a manager sees with access, and who the pickers offer.
  "access.grant.changed": [getTeamKey, getProjectAccessKey, getProjectPeopleKey],
  "role.updated": [getWorkspaceRolesKey],
};

// Topics that can change what someone may do; followPermissionChange works out whether that someone is the viewer.
const permissionTopics = new Set([
  "workspace.member.added",
  "workspace.member.removed",
  "workspace.member.updated",
  "role.updated",
  "access.grant.changed",
]);

const heldPermissions = (client: QueryClient) =>
  JSON.stringify([
    client.getQueriesData<MyWorkspaceInfo>({ queryKey: [getMyRoleKey] }).map(([, data]) => [data?.permissions, data?.projects]),
    client.getQueryData<MeResponse>([getMeKey])?.instance_permissions,
  ]);

// Every read is checked on the server, so once the viewer's permissions move, all open reads refetch through those checks.
const followPermissionChange = async (client: QueryClient, frame: ServerFrame) => {
  const { user_id: userID } = frame.payload as { user_id?: string };
  if (userID && userID === client.getQueryData<MeResponse>([getMeKey])?.user.id) {
    await client.invalidateQueries();
    return;
  }
  // A role frame names no holder, so the viewer's own permissions are refetched and compared.
  if (frame.topic !== "role.updated") return;
  const before = heldPermissions(client);
  await Promise.all([client.refetchQueries({ queryKey: [getMyRoleKey] }), client.refetchQueries({ queryKey: [getMeKey] })]);
  if (heldPermissions(client) === before) return;
  await client.invalidateQueries();
};

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

interface RouterFollowers {
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

const dispatch = (client: ReturnType<typeof useQueryClient>, router: RouterFollowers) => (frame: ServerFrame) => {
  if (frame.topic === "workspace.updated") {
    router.onWorkspaceUpdated(frame.payload as WorkspaceUpdate);
    return;
  }
  if (frame.topic === "chat.conversation.deleted") router.onConversationDeleted(frame.payload as ConversationDeleted);
  if (frame.topic === "topology") {
    const flow = useFlowStore.getState();
    const parsed = CanvasFrameSchema.safeParse(frame.payload);
    if (parsed.success && parsed.data.workspace_id === flow.workspaceId) flow.applyServerPatch(parsed.data);
    return;
  }
  if (frame.topic === "chat.agent.stream") {
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
    return;
  }
  if (frame.topic === "play.run") {
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
    // Starting shows the Thread section's Started message in place of "Start chat" without a reload.
    if (p.state === "starting") void client.invalidateQueries({ queryKey: [getChatTicketThreadStatusKey, p.target_id] });
    // A done run's new column and linked branch/PR show up without a reload.
    if (!isTrailActive(p.state)) {
      void client.invalidateQueries({ queryKey: [getTicketsKey] });
      void client.invalidateQueries({ queryKey: [getTicketKey, p.target_id] });
      void client.invalidateQueries({ queryKey: [getTicketLinksKey, p.target_id] });
    }
    return;
  }
  if (frame.topic === "chat.message.created" || frame.topic === "chat.message.updated") {
    const p = frame.payload as MessagePayload;
    if (p.message) upsertCachedMessage(client, p.message);
    // Once the turn's real message lands (author_kind "agent"), the ephemeral stream bubble yields to it; a note does not end it.
    if (p.message?.author_kind === "agent" && !isNote(p.message)) yieldStream(p.message);
    // A note's file changed under its message: open renders and the pill's size follow it; an open editor follows its room.
    if (frame.topic === "chat.message.updated" && p.message && isNote(p.message)) {
      void client.invalidateQueries({ queryKey: [getNoteTextKey, p.message.attachment_id] });
      void client.invalidateQueries({ queryKey: [getAttachmentsKey, { conversation_id: p.message.conversation_id }] });
    }
    if (frame.topic === "chat.message.updated") return;
  }
  if (frame.topic === "chat.message.deleted") {
    const p = frame.payload as MessageDeletedPayload;
    if (p.message_id) markCachedMessageDeleted(client, p.conversation_id, p.message_id, p.deleted_at);
  }
  if (frame.topic === "chat.message.reactions_changed") {
    const p = frame.payload as MessageReactionsChangedPayload;
    applyCachedReaction(client, p.conversation_id, { messageId: p.message_id, userId: p.user_id, emoji: p.emoji, reacted: p.reacted });
    return;
  }
  if (frame.topic === "computer.tunnel_status_changed") {
    setCachedTunnelStatus(client, frame.payload as TunnelStatusChangedPayload);
    return;
  }
  if (frame.topic === "computer.setup_turn_activity") {
    const p = frame.payload as SetupTurnActivityPayload;
    useSetupActivityStore
      .getState()
      .push(p.turn_id, { kind: p.kind ?? "other", call_id: p.call_id ?? "", tool: p.tool ?? "", summary: p.status, detail: p.text ?? "", at: p.at ?? "" });
    return;
  }
  if (frame.topic === "session.created") {
    const p = frame.payload as SessionCreatedPayload;
    // Every browser hears every session; only the viewer's own phone flips their Devices page.
    const me = client.getQueryData<MeResponse>([getMeKey]);
    if (p.client === "phone" && p.user_id === me?.user.id) {
      useDeviceArrivalStore.getState().arrive({ id: p.session_id, platform: p.platform, label: p.label });
    }
  }
  if (frame.topic === "voice.occupancy.changed") {
    const p = frame.payload as OccupancyChangedPayload;
    useVoiceOccupancyStore.getState().setChannel(p.conversation_id, p.occupants);
    return;
  }
  if (permissionTopics.has(frame.topic)) void followPermissionChange(client, frame);
  const keys = pushTopics[frame.topic];
  if (keys) {
    keys.forEach((key) => void client.invalidateQueries({ queryKey: [key] }));
    return;
  }
  void client.invalidateQueries({ queryKey: [frame.topic] });
};

export const useLiveEvents = (url: string | null, opts: LiveEventsClientOptions = {}) => {
  const client = useQueryClient();
  const optsRef = useRef(opts);
  const navigate = useNavigate();
  const location = useLocation();
  // The socket outlives every render, so a workspace rename reads the router's latest state from here.
  const routerRef = useRef({ navigate, location });
  useEffect(() => {
    routerRef.current = { navigate, location };
  });

  useEffect(() => {
    if (!url) return;
    const events = new LiveEventsClient(url, {
      ...optsRef.current,
      onReconnect: () => {
        optsRef.current.onReconnect?.();
        void notifyIfServerUpdated(client);
      },
    });
    const unsubscribe = events.subscribe(
      dispatch(client, {
        onWorkspaceUpdated: (update) => followWorkspaceUpdate(client, routerRef.current.navigate, routerRef.current.location, update),
        onConversationDeleted: (deleted) => followConversationDeleted(routerRef.current.navigate, routerRef.current.location, deleted),
      }),
    );
    events.connect();
    return () => {
      unsubscribe();
      events.close();
    };
  }, [url, client]);
};
