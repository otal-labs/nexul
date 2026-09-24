import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { CanvasSchema } from "@/models/Topology";
import {
  LiveEventsClient,
  type LiveEventsClientOptions,
  type ServerFrame,
} from "@/api/ws";
import { useFlowStore } from "@/stores/flowStore";
import { getDeployKey, getDeployLogKey } from "@/hooks/DeployHooks";
import { getDnsExposuresKey, getDnsGatewaysKey } from "@/hooks/DnsHooks";
import { getDocKey, getDocsKey } from "@/hooks/DocHooks";
import { getInstanceUpgradeKey } from "@/hooks/InstanceUpgradeHooks";
import { getNotificationsKey, getUnreadCountKey } from "@/hooks/NotificationHooks";
import { getRunnerQueueKey, getRunnersKey } from "@/hooks/RunnerHooks";
import { getServiceDeploysKey, getServicesKey } from "@/hooks/ServiceHooks";
import { getStackDeploysKey } from "@/hooks/StackHooks";
import { getCategoriesKey, getProjectCategoriesKey } from "@/hooks/CategoryHooks";
import { getProjectTicketTypesKey, getTicketTypesKey } from "@/hooks/TicketTypeHooks";
import { getProjectStatusesKey, getStatusesKey } from "@/hooks/StatusHooks";
import { getTicketKey, getTicketLinksKey, getTicketsKey } from "@/hooks/TicketHooks";
import {
  getChatConversationsKey,
  getChatTicketThreadStatusKey,
  getChatUnreadKey,
  markCachedMessageDeleted,
  upsertCachedMessage,
} from "@/hooks/ChatHooks";
import type { Message } from "@/models/Chat";
import { getServerVersionKey, notifyIfServerUpdated } from "@/hooks/VersionHooks";
import { getApplicablePlaysKey, getWorkspacePlaysKey } from "@/hooks/PlayHooks";
import { getActiveTrailsKey, getTrailKey, getTrailsKey } from "@/hooks/TrailHooks";
import { useAgentStreamStore } from "@/stores/agentStreamStore";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useVoiceOccupancyStore } from "@/stores/voiceOccupancyStore";
import { isTrailActive, type ActivityKind, type RunFrame } from "@/models/Trail";
import type { VoiceOccupant } from "@/models/Voice";

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
  "category.created": [getCategoriesKey, getProjectCategoriesKey],
  "category.updated": [getCategoriesKey, getProjectCategoriesKey],
  "category.deleted": [getCategoriesKey, getProjectCategoriesKey],
  "doc.created": [getDocsKey],
  "doc.updated": [getDocsKey, getDocKey],
  "ticket.created": [getTicketsKey],
  "ticket.updated": [getTicketsKey, getTicketKey, getTicketLinksKey, getTrailsKey],
  "ticket.status_changed": [getTicketsKey, getTicketKey, getTicketLinksKey, getTrailsKey],
  "ticket.developer_changed": [getTicketsKey, getTicketKey],
  "ticket.tester_changed": [getTicketsKey, getTicketKey],
  "ticket.finished": [getTicketsKey],
  "ticket.category_changed": [getTicketsKey],
  "ticket_type.created": [getTicketTypesKey, getProjectTicketTypesKey],
  "ticket_type.updated": [getTicketTypesKey, getProjectTicketTypesKey],
  "ticket_type.deleted": [getTicketTypesKey, getProjectTicketTypesKey],
  "status.created": [getStatusesKey, getProjectStatusesKey, getTicketsKey],
  "status.updated": [getStatusesKey, getProjectStatusesKey, getTicketsKey],
  "status.deleted": [getStatusesKey, getProjectStatusesKey, getTicketsKey],
  "chat.conversation.created": [getChatConversationsKey],
  "chat.message.created": [getChatConversationsKey, getChatUnreadKey],
  "chat.message.deleted": [getChatUnreadKey],
  "instance.upgrade_changed": [getInstanceUpgradeKey, getServerVersionKey],
  "play.created": [getWorkspacePlaysKey, getApplicablePlaysKey],
  "play.updated": [getWorkspacePlaysKey, getApplicablePlaysKey],
  "play.deleted": [getWorkspacePlaysKey, getApplicablePlaysKey],
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

// One voice channel's full occupant list after a change, applied wholesale.
interface OccupancyChangedPayload {
  conversation_id: string;
  occupants: VoiceOccupant[];
}

const dispatch = (client: ReturnType<typeof useQueryClient>) => (frame: ServerFrame) => {
  if (frame.topic === "topology") {
    const parsed = CanvasSchema.safeParse(frame.payload);
    if (parsed.success) useFlowStore.getState().applyServerPatch(parsed.data);
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
    // Once the turn's real message lands (author_kind "agent"), the ephemeral stream bubble yields to it.
    if (p.message?.author_kind === "agent") useAgentStreamStore.getState().clearStream(p.message.conversation_id);
    if (frame.topic === "chat.message.updated") return;
  }
  if (frame.topic === "chat.message.deleted") {
    const p = frame.payload as MessageDeletedPayload;
    if (p.message_id) markCachedMessageDeleted(client, p.conversation_id, p.message_id, p.deleted_at);
  }
  if (frame.topic === "voice.occupancy.changed") {
    const p = frame.payload as OccupancyChangedPayload;
    useVoiceOccupancyStore.getState().setChannel(p.conversation_id, p.occupants);
    return;
  }
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

  useEffect(() => {
    if (!url) return;
    const events = new LiveEventsClient(url, {
      ...optsRef.current,
      onReconnect: () => {
        optsRef.current.onReconnect?.();
        void notifyIfServerUpdated(client);
      },
    });
    const unsubscribe = events.subscribe(dispatch(client));
    events.connect();
    return () => {
      unsubscribe();
      events.close();
    };
  }, [url, client]);
};
