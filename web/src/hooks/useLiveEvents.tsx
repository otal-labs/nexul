import { useEffect, useRef } from "react";
import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router";

import { LiveEventsClient, type LiveEventsClientOptions, type ServerFrame } from "@/api/ws";
import { frameHandlers, type RouterFollowers } from "@/hooks/liveFrameHandlers";
import { pushTopics } from "@/hooks/livePushTopics";
import { followWorkspaceUpdate, getMyRoleKey } from "@/hooks/WorkspaceHooks";
import { getMeKey } from "@/hooks/AuthHooks";
import { followConversationDeleted } from "@/hooks/ChatHooks";
import { notifyIfServerUpdated } from "@/hooks/VersionHooks";
import type { MyWorkspaceInfo } from "@/models/Permission";
import type { MeResponse } from "@/models/User";

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

const dispatch = (client: QueryClient, router: RouterFollowers) => (frame: ServerFrame) => {
  frameHandlers[frame.topic]?.(frame, client, router);
  if (permissionTopics.has(frame.topic)) void followPermissionChange(client, frame);
  pushTopics[frame.topic]?.forEach((key) => void client.invalidateQueries({ queryKey: [key] }));
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
