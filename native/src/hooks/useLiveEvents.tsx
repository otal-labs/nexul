import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { AppState, type AppStateStatus } from "react-native";

import { buildLiveURL, LiveEventsClient, type ServerFrame } from "@/api/events";
import { getMeKey } from "@/hooks/AuthHooks";
import { getChatConversationsKey, getChatMessagesKey, getChatUnreadKey } from "@/hooks/ChatHooks";
import { getDeployKey, getDeployLogKey } from "@/hooks/DeployHooks";
import { getDocKey, getDocsKey } from "@/hooks/DocHooks";
import { getNotificationsKey, getUnreadCountKey } from "@/hooks/NotificationHooks";
import { getWorkspacePeopleKey } from "@/hooks/PeopleHooks";
import { getRunnersKey } from "@/hooks/RunnerHooks";
import { getStackDeploysKey } from "@/hooks/StackHooks";
import { getProjectStatusesKey } from "@/hooks/StatusHooks";
import { getTicketKey, getTicketsByProjectKey } from "@/hooks/TicketHooks";
import type { MeResponse } from "@/models/User";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

// Maps push topics to the query keys they invalidate; each domain adds its rows as its screens land.
const pushTopics: Record<string, string[]> = {
  "notification.created": [getNotificationsKey, getUnreadCountKey],
  "chat.conversation.created": [getChatConversationsKey],
  "chat.conversation.updated": [getChatConversationsKey],
  // An open thread of a deleted channel refetches into its error state instead of showing stale messages.
  "chat.conversation.deleted": [getChatConversationsKey, getChatMessagesKey, getChatUnreadKey],
  // A private channel the viewer lost drops from the list, and its open thread refetches into not found.
  "chat.conversation.members_changed": [getChatConversationsKey, getChatMessagesKey, getChatUnreadKey],
  "account.profile_updated": [getWorkspacePeopleKey],
  "account.removed": [getWorkspacePeopleKey],
  "workspace.member.added": [getWorkspacePeopleKey],
  "workspace.member.removed": [getWorkspacePeopleKey],
  "chat.message.created": [getChatMessagesKey, getChatUnreadKey],
  "chat.message.updated": [getChatMessagesKey],
  "chat.message.deleted": [getChatMessagesKey, getChatUnreadKey],
  "ticket.created": [getTicketsByProjectKey],
  "ticket.updated": [getTicketsByProjectKey, getTicketKey],
  "ticket.status_changed": [getTicketsByProjectKey, getTicketKey],
  "ticket.assignee_changed": [getTicketsByProjectKey, getTicketKey],
  "ticket.developer_changed": [getTicketsByProjectKey, getTicketKey],
  "ticket.tester_changed": [getTicketsByProjectKey, getTicketKey],
  "ticket.finished": [getTicketsByProjectKey, getTicketKey],
  "ticket.deleted": [getTicketsByProjectKey, getTicketKey],
  "status.created": [getProjectStatusesKey],
  "status.updated": [getProjectStatusesKey],
  "status.deleted": [getProjectStatusesKey],
  "doc.created": [getDocsKey],
  "doc.updated": [getDocsKey, getDocKey],
  "doc.deleted": [getDocsKey, getDocKey],
  "runner.connected": [getRunnersKey],
  "runner.disconnected": [getRunnersKey],
  // The deploy screen's log follows the tail off this alone: a refetch here is what turns into new log lines.
  "deploy.updated": [getDeployKey, getDeployLogKey, getStackDeploysKey],
};

// Topics that can change what the person named as user_id may do, Project access included (ADR 0097).
const permissionTopics = new Set([
  "workspace.member.added",
  "workspace.member.removed",
  "workspace.member.updated",
  "access.grant.changed",
]);

// Once the viewer's own access moves every open read refetches, so a project taken away turns its open screen revoked.
const isViewersAccessChange = (client: QueryClient, frame: ServerFrame) => {
  if (!permissionTopics.has(frame.topic)) return false;
  const userID = (frame.payload as { user_id?: string } | null)?.user_id;
  return !!userID && userID === client.getQueryData<MeResponse>([getMeKey])?.user.id;
};

export const dispatch = (client: QueryClient) => (frame: ServerFrame) => {
  if (isViewersAccessChange(client, frame)) {
    void client.invalidateQueries();
    return;
  }
  pushTopics[frame.topic]?.forEach((key) => void client.invalidateQueries({ queryKey: [key] }));
};

// One socket for the signed-in instance: open while the app is in front, closed in the background.
export const useLiveEvents = () => {
  const client = useQueryClient();
  const host = useSessionStore((s) => (s.signedIn ? s.host : null));

  useEffect(() => {
    if (!host) return;
    let events: LiveEventsClient | null = null;
    const open = () => {
      const token = readSessionToken();
      if (events || !token) return;
      events = new LiveEventsClient(buildLiveURL(host, token));
      events.subscribe(dispatch(client));
      events.connect();
    };
    const close = () => {
      events?.close();
      events = null;
    };
    const onAppState = (status: AppStateStatus) => {
      if (status === "active") {
        open();
        return;
      }
      close();
    };
    onAppState(AppState.currentState);
    const subscription = AppState.addEventListener("change", onAppState);
    return () => {
      subscription.remove();
      close();
    };
  }, [host, client]);
};
