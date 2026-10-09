import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { AppState, type AppStateStatus } from "react-native";

import { buildLiveURL, LiveEventsClient, type ServerFrame } from "@/api/events";
import { getMeKey } from "@/hooks/AuthHooks";
import { applyRefresh, liveQueries } from "@/lib/liveQuery";
import type { MeResponse } from "@/models/User";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

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

// Each query's definition names the topics that refresh it and how (ADR 0136); a frame reaches exactly those.
export const dispatch = (client: QueryClient) => (frame: ServerFrame) => {
  if (isViewersAccessChange(client, frame)) {
    void client.invalidateQueries();
    return;
  }
  for (const query of liveQueries()) {
    const refresh = query.refreshes[frame.topic];
    if (refresh) applyRefresh(client, query.key, refresh, frame.payload);
  }
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
