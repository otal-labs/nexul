import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { AppState, type AppStateStatus } from "react-native";

import { buildLiveURL, LiveEventsClient, type ServerFrame } from "@/api/events";
import { getChatConversationsKey, getChatMessagesKey, getChatUnreadKey } from "@/hooks/ChatHooks";
import { getDocKey, getDocsKey } from "@/hooks/DocHooks";
import { getNotificationsKey, getUnreadCountKey } from "@/hooks/NotificationHooks";
import { getProjectStatusesKey } from "@/hooks/StatusHooks";
import { getTicketKey, getTicketsByProjectKey } from "@/hooks/TicketHooks";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

// Maps push topics to the query keys they invalidate; each domain adds its rows as its screens land.
const pushTopics: Record<string, string[]> = {
  "notification.created": [getNotificationsKey, getUnreadCountKey],
  "chat.conversation.created": [getChatConversationsKey],
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
};

export const dispatch = (client: QueryClient) => (frame: ServerFrame) => {
  const keys = pushTopics[frame.topic];
  if (keys) {
    keys.forEach((key) => void client.invalidateQueries({ queryKey: [key] }));
    return;
  }
  void client.invalidateQueries({ queryKey: [frame.topic] });
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
