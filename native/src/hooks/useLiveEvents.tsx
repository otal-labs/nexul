import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { AppState, type AppStateStatus } from "react-native";

import { LiveEventsClient, type ServerFrame } from "@nexul/client-core/liveSocket";
import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

import { buildLiveURL } from "@/api/events";
import { getMeKey } from "@/hooks/AuthHooks";
import { getMyRoleKey } from "@/hooks/WorkspaceHooks";
import { applyRefresh, liveQueries } from "@/lib/liveQuery";
import type { MeResponse } from "@/models/User";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

// Topics that can change what the person named as user_id, or the holders of the role named as role_id, may do.
const permissionTopics = new Set([
  "workspace.member.added",
  "workspace.member.removed",
  "workspace.member.updated",
  "access.grant.changed",
]);

// Once the viewer's own access moves every open read refetches, so a project taken away turns its open screen revoked.
const isViewersAccessChange = (client: QueryClient, frame: ServerFrame) => {
  if (!permissionTopics.has(frame.topic)) return false;
  const { user_id: userID, role_id: roleID } = (frame.payload ?? {}) as { user_id?: string; role_id?: string };
  if (userID) return userID === client.getQueryData<MeResponse>([getMeKey])?.user.id;
  if (!roleID) return false;
  // A role rule also reaches people who manage the entity; only its holders' access moved.
  return client
    .getQueriesData<MyWorkspaceInfo>({ queryKey: [getMyRoleKey] })
    .some(([, info]) => info?.roles?.some((role) => role.id === roleID));
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
      // Frames sent while it was down are lost, so every open read refetches; the focus manager covers a foreground.
      events = new LiveEventsClient(buildLiveURL(host, token), { onReconnect: () => void client.invalidateQueries() });
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
